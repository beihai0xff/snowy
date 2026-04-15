package http

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/agent"
	"github.com/beihai0xff/snowy/internal/handler/http/dto"
	"github.com/beihai0xff/snowy/internal/pkg/common"
)

// AgentHandler Agent 会话 HTTP Handler。
// 参考技术方案 §17.1 & §17.6。
type AgentHandler struct {
	agentSvc    agent.Service
	writeSvc    agent.WriteService
	sessionRepo agent.SessionRepository
	messageRepo agent.MessageRepository
}

// NewAgentHandler 创建 AgentHandler。
func NewAgentHandler(
	agentSvc agent.Service,
	writeSvc agent.WriteService,
	sessionRepo agent.SessionRepository,
	messageRepo agent.MessageRepository,
) *AgentHandler {
	return &AgentHandler{
		agentSvc:    agentSvc,
		writeSvc:    writeSvc,
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
	}
}

// Chat POST /api/v1/agent/chat — 统一会话入口（支持 SSE）。
func (h *AgentHandler) Chat(c *gin.Context) {
	var req dto.ChatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	// 判断是否请求 SSE 流式
	if c.GetHeader("Accept") == "text/event-stream" {
		h.chatStream(c, &req)

		return
	}

	chatReq := &agent.ChatRequest{
		SessionID: parseOptionalUUID(req.SessionID),
		Message:   req.Message,
		Mode:      agent.Mode(req.Mode),
		Filters: agent.Filters{
			Subject: req.Filters.Subject,
			Grade:   req.Filters.Grade,
		},
	}

	resp, err := h.agentSvc.Chat(c.Request.Context(), chatReq)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	if h.writeSvc != nil {
		persisted, persistErr := h.writeSvc.PersistConversation(c.Request.Context(), &agent.PersistConversationInput{
			UserID:    parseOptionalUUID(common.UserIDFromContext(c.Request.Context())),
			SessionID: chatReq.SessionID,
			Mode:      chatReq.Mode,
			Message:   chatReq.Message,
			Filters:   chatReq.Filters,
			Response:  resp,
		})
		if persistErr != nil {
			reqID := common.RequestIDFromContext(c.Request.Context())
			c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

			return
		}

		if persisted != nil && persisted.Session != nil {
			c.Header("X-Session-ID", persisted.Session.ID.String())
		}
	}

	c.JSON(http.StatusOK, common.Success(resp))
}

// CreateSession POST /api/v1/agent/sessions — 创建会话。
func (h *AgentHandler) CreateSession(c *gin.Context) {
	var req dto.CreateSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage(err.Error()), reqID))

		return
	}

	uid := parseOptionalUUID(common.UserIDFromContext(c.Request.Context()))

	var session *agent.Session
	var err error
	if h.writeSvc != nil {
		session, err = h.writeSvc.CreateSession(c.Request.Context(), &agent.CreateSessionInput{
			UserID: uid,
			Mode:   agent.Mode(req.Mode),
		})
	} else {
		session = &agent.Session{
			ID:        uuid.New(),
			UserID:    uid,
			Mode:      agent.Mode(req.Mode),
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = h.sessionRepo.Create(c.Request.Context(), session)
	}

	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	c.JSON(http.StatusCreated, common.Success(dto.SessionResp{
		ID:        session.ID,
		Mode:      string(session.Mode),
		Status:    session.Status,
		CreatedAt: session.CreatedAt.Format(time.RFC3339),
	}))
}

// GetSession GET /api/v1/agent/sessions/:id — 获取会话详情。
func (h *AgentHandler) GetSession(c *gin.Context) {
	idStr := c.Param("id")

	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage("invalid session id"), reqID))

		return
	}

	session, err := h.sessionRepo.GetByID(c.Request.Context(), sessionID)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusNotFound, common.Fail(common.ErrInvalidInput.WithMessage("session not found"), reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(dto.SessionResp{
		ID:        session.ID,
		Mode:      string(session.Mode),
		Status:    session.Status,
		CreatedAt: session.CreatedAt.Format(time.RFC3339),
	}))
}

// ListMessages GET /api/v1/agent/sessions/:id/messages — 获取会话消息列表。
func (h *AgentHandler) ListMessages(c *gin.Context) {
	idStr := c.Param("id")

	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusBadRequest, common.Fail(common.ErrInvalidInput.WithMessage("invalid session id"), reqID))

		return
	}

	msgs, total, err := h.messageRepo.ListBySession(c.Request.Context(), sessionID, 0, 50)
	if err != nil {
		reqID := common.RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusInternalServerError, common.Fail(common.ErrInternal, reqID))

		return
	}

	c.JSON(http.StatusOK, common.Success(common.PageResponse{
		Total:    total,
		Page:     1,
		PageSize: 50,
		Items:    msgs,
	}))
}

// chatStream 流式输出 SSE。
func (h *AgentHandler) chatStream(c *gin.Context, req *dto.ChatReq) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	events := make(chan agent.SSEEvent, 32)
	errCh := make(chan error, 1)

	chatReq := buildChatRequest(req)

	go func() {
		defer close(events)

		errCh <- h.agentSvc.ChatStream(c.Request.Context(), chatReq, events)
	}()

	aggregator := agent.NewStreamResponseAggregator(chatReq.Mode)

	disconnected := c.Stream(func(_ io.Writer) bool {
		event, ok := <-events
		if !ok {
			return false
		}

		if err := aggregator.Consume(event); err != nil {
			slog.WarnContext(c.Request.Context(), "consume stream event failed", "error", err, "event", event.Event)
		}

		c.SSEvent(string(event.Event), event.Data)

		return true
	})

	streamErr := <-errCh
	if shouldSkipStreamPersistence(disconnected, streamErr, aggregator, h.writeSvc != nil, c.Request.Context().Err()) {
		if streamErr != nil {
			slog.WarnContext(c.Request.Context(), "agent stream ended without persistence", "error", streamErr)
		}

		return
	}

	persisted, err := h.writeSvc.PersistConversation(c.Request.Context(), &agent.PersistConversationInput{
		UserID:    parseOptionalUUID(common.UserIDFromContext(c.Request.Context())),
		SessionID: chatReq.SessionID,
		Mode:      chatReq.Mode,
		Message:   chatReq.Message,
		Filters:   chatReq.Filters,
		Response:  aggregator.Response(),
	})
	if err != nil {
		slog.WarnContext(c.Request.Context(), "persist streamed conversation failed", "error", err)
		return
	}

	if persisted != nil && persisted.Session != nil {
		c.Header("X-Session-ID", persisted.Session.ID.String())
	}
}

func parseOptionalUUID(raw string) uuid.UUID {
	if raw == "" {
		return uuid.Nil
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil
	}

	return id
}

func buildChatRequest(req *dto.ChatReq) *agent.ChatRequest {
	return &agent.ChatRequest{
		SessionID: parseOptionalUUID(req.SessionID),
		Message:   req.Message,
		Mode:      agent.Mode(req.Mode),
		Filters: agent.Filters{
			Subject: req.Filters.Subject,
			Grade:   req.Filters.Grade,
		},
	}
}

func shouldSkipStreamPersistence(
	disconnected bool,
	streamErr error,
	aggregator *agent.StreamResponseAggregator,
	hasWriteService bool,
	requestErr error,
) bool {
	return disconnected || streamErr != nil || !aggregator.Done() || !hasWriteService || requestErr != nil
}
