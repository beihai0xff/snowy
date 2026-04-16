package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	irepo "github.com/beihai0xff/snowy/internal/repo"
)

// WriteService 负责 Agent 会话写路径的原子持久化。
type WriteService interface {
	CreateSession(ctx context.Context, input *CreateSessionInput) (*Session, error)
	PersistConversation(ctx context.Context, input *PersistConversationInput) (*PersistConversationResult, error)
}

// CreateSessionInput 创建会话所需参数。
type CreateSessionInput struct {
	UserID   uuid.UUID
	Mode     Mode
	Metadata map[string]any
}

// PersistConversationInput 一次对话写入所需参数。
type PersistConversationInput struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Mode      Mode
	Message   string
	Filters   Filters
	Response  *ChatResponse
}

// PersistConversationResult 返回事务内生成的记录。
type PersistConversationResult struct {
	Session          *Session
	UserMessage      *Message
	AssistantMessage *Message
	Run              *Run
	ToolCalls        []*RunToolCall
}

type writeService struct {
	transactor   irepo.Transactor
	sessionRepo  SessionRepository
	messageRepo  MessageRepository
	runRepo      RunRepository
	toolCallRepo ToolCallRepository
}

// NewWriteService 创建 Agent 写路径服务。
func NewWriteService(
	transactor irepo.Transactor,
	sessionRepo SessionRepository,
	messageRepo MessageRepository,
	runRepo RunRepository,
	toolCallRepo ToolCallRepository,
) WriteService {
	return &writeService{
		transactor:   transactor,
		sessionRepo:  sessionRepo,
		messageRepo:  messageRepo,
		runRepo:      runRepo,
		toolCallRepo: toolCallRepo,
	}
}

func (s *writeService) CreateSession(ctx context.Context, input *CreateSessionInput) (*Session, error) {
	if input == nil {
		return nil, errors.New("create session input is nil")
	}

	mode := input.Mode
	if mode == "" {
		mode = ModeAuto
	}

	now := time.Now()
	session := &Session{
		ID:        uuid.New(),
		UserID:    input.UserID,
		Mode:      mode,
		Status:    "active",
		Metadata:  input.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.withTransaction(ctx, func(txCtx context.Context) error {
		return s.sessionRepo.Create(txCtx, session)
	}); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return session, nil
}

//nolint:cyclop // The transaction persists the full conversation atomically in one place.
func (s *writeService) PersistConversation(
	ctx context.Context,
	input *PersistConversationInput,
) (*PersistConversationResult, error) {
	if input == nil {
		return nil, errors.New("persist conversation input is nil")
	}

	if input.Response == nil {
		return nil, errors.New("persist conversation response is nil")
	}

	if input.Message == "" {
		return nil, errors.New("persist conversation message is empty")
	}

	result := &PersistConversationResult{}

	if err := s.withTransaction(ctx, func(txCtx context.Context) error {
		now := time.Now()

		result.Session = s.buildSession(input, now)
		if input.SessionID == uuid.Nil {
			if err := s.sessionRepo.Create(txCtx, result.Session); err != nil {
				return fmt.Errorf("create session: %w", err)
			}
		}

		result.UserMessage = &Message{
			ID:        uuid.New(),
			SessionID: result.Session.ID,
			Role:      "user",
			Content:   input.Message,
			CreatedAt: now,
		}
		if err := s.messageRepo.Save(txCtx, result.UserMessage); err != nil {
			return fmt.Errorf("save user message: %w", err)
		}

		result.AssistantMessage = &Message{
			ID:        uuid.New(),
			SessionID: result.Session.ID,
			Role:      "assistant",
			Content:   input.Response.Answer,
			CreatedAt: now,
		}
		if err := s.messageRepo.Save(txCtx, result.AssistantMessage); err != nil {
			return fmt.Errorf("save assistant message: %w", err)
		}

		result.Run = &Run{
			ID:            uuid.New(),
			SessionID:     result.Session.ID,
			MessageID:     result.UserMessage.ID,
			Mode:          s.resolveMode(input),
			ModelName:     "",
			PromptVersion: "",
			Confidence:    input.Response.Confidence,
			Status:        "success",
			CreatedAt:     now,
		}
		if err := s.runRepo.Save(txCtx, result.Run); err != nil {
			return fmt.Errorf("save agent run: %w", err)
		}

		result.ToolCalls = make([]*RunToolCall, 0, len(input.Response.ToolCalls))
		for _, call := range input.Response.ToolCalls {
			tc := &RunToolCall{
				ID:        uuid.New(),
				RunID:     result.Run.ID,
				ToolName:  call.Tool,
				Status:    call.Status,
				CreatedAt: now,
			}
			if err := s.toolCallRepo.Save(txCtx, tc); err != nil {
				return fmt.Errorf("save agent tool call: %w", err)
			}

			result.ToolCalls = append(result.ToolCalls, tc)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("persist conversation: %w", err)
	}

	return result, nil
}

func (s *writeService) buildSession(input *PersistConversationInput, now time.Time) *Session {
	metadata := map[string]any{}
	if input.Filters.Subject != "" {
		metadata["subject"] = input.Filters.Subject
	}

	if input.Filters.Grade != "" {
		metadata["grade"] = input.Filters.Grade
	}

	if len(metadata) == 0 {
		metadata = nil
	}

	id := input.SessionID
	if id == uuid.Nil {
		id = uuid.New()
	}

	return &Session{
		ID:        id,
		UserID:    input.UserID,
		Mode:      s.resolveMode(input),
		Status:    "active",
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (s *writeService) resolveMode(input *PersistConversationInput) Mode {
	if input.Response != nil && input.Response.Mode != "" {
		return input.Response.Mode
	}

	if input.Mode != "" {
		return input.Mode
	}

	return ModeAuto
}

func (s *writeService) withTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if s.transactor == nil {
		return fn(ctx)
	}

	return s.transactor.Transaction(ctx, fn)
}
