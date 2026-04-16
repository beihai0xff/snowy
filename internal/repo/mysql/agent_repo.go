package mysql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/beihai0xff/snowy/internal/agent"
)

// agentSessionRepo 实现 agent.SessionRepository 接口。
type agentSessionRepo struct {
	db *gorm.DB
}

// NewAgentSessionRepository 创建 Agent Session Repository。
func NewAgentSessionRepository(db *gorm.DB) agent.SessionRepository {
	return &agentSessionRepo{db: db}
}

func (r *agentSessionRepo) Create(ctx context.Context, s *agent.Session) error {
	err := dbFromContext(ctx, r.db).Create(newAgentSessionRow(s)).Error
	if err != nil {
		return fmt.Errorf("insert agent session: %w", err)
	}

	return nil
}

func (r *agentSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*agent.Session, error) {
	row := &agentSessionRow{}

	err := dbFromContext(ctx, r.db).Where("id = ?", id).Take(row).Error
	if err != nil {
		return nil, fmt.Errorf("get agent session: %w", err)
	}

	return row.toDomain(), nil
}

func (r *agentSessionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	err := dbFromContext(ctx, r.db).
		Model(&agentSessionRow{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     status,
			"updated_at": gorm.Expr("NOW(3)"),
		}).Error
	if err != nil {
		return fmt.Errorf("update agent session status: %w", err)
	}

	return nil
}

func (r *agentSessionRepo) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*agent.Session, int64, error) {
	return listByUserRows[agentSessionRow](ctx, r.db, &agentSessionRow{}, userID, offset, limit,
		"created_at DESC",
		"agent sessions", "agent sessions",
		func(row *agentSessionRow) (*agent.Session, error) {
			return row.toDomain(), nil
		},
	)
}

// agentRunRepo 实现 agent.RunRepository 接口。
type agentRunRepo struct {
	db *gorm.DB
}

// NewAgentRunRepository 创建 Agent Run Repository。
func NewAgentRunRepository(db *gorm.DB) agent.RunRepository {
	return &agentRunRepo{db: db}
}

func (r *agentRunRepo) Save(ctx context.Context, run *agent.Run) error {
	if err := dbFromContext(ctx, r.db).Create(newAgentRunRow(run)).Error; err != nil {
		return fmt.Errorf("insert agent run: %w", err)
	}

	return nil
}

func (r *agentRunRepo) GetByID(ctx context.Context, id uuid.UUID) (*agent.Run, error) {
	row := &agentRunRow{}

	err := dbFromContext(ctx, r.db).Where("id = ?", id).Take(row).Error
	if err != nil {
		return nil, fmt.Errorf("get agent run: %w", err)
	}

	return row.toDomain(), nil
}

// agentMessageRepo 实现 agent.MessageRepository 接口。
type agentMessageRepo struct {
	db *gorm.DB
}

// NewAgentMessageRepository 创建 Agent Message Repository。
func NewAgentMessageRepository(db *gorm.DB) agent.MessageRepository {
	return &agentMessageRepo{db: db}
}

func (r *agentMessageRepo) Save(ctx context.Context, msg *agent.Message) error {
	if err := dbFromContext(ctx, r.db).Create(newAgentMessageRow(msg)).Error; err != nil {
		return fmt.Errorf("insert agent message: %w", err)
	}

	return nil
}

func (r *agentMessageRepo) ListBySession(
	ctx context.Context,
	sessionID uuid.UUID,
	offset, limit int,
) ([]*agent.Message, int64, error) {
	var total int64

	err := dbFromContext(ctx, r.db).Model(&agentMessageRow{}).Where("session_id = ?", sessionID).Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("count agent messages: %w", err)
	}

	rows := make([]agentMessageRow, 0, limit)
	// 同毫秒时间戳下按 user -> assistant -> system 顺序返回，保证会话回放稳定。
	err = dbFromContext(ctx, r.db).
		Where("session_id = ?", sessionID).
		Order("created_at ASC, FIELD(role, 'user', 'assistant', 'system') ASC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list agent messages: %w", err)
	}

	msgs := make([]*agent.Message, 0, len(rows))
	for i := range rows {
		msgs = append(msgs, rows[i].toDomain())
	}

	return msgs, total, nil
}

// agentToolCallRepo 实现 agent.ToolCallRepository 接口。
type agentToolCallRepo struct {
	db *gorm.DB
}

// NewAgentToolCallRepository 创建 Agent ToolCall Repository。
func NewAgentToolCallRepository(db *gorm.DB) agent.ToolCallRepository {
	return &agentToolCallRepo{db: db}
}

func (r *agentToolCallRepo) Save(ctx context.Context, tc *agent.RunToolCall) error {
	if err := dbFromContext(ctx, r.db).Create(newAgentToolCallRow(tc)).Error; err != nil {
		return fmt.Errorf("insert agent tool call: %w", err)
	}

	return nil
}

func (r *agentToolCallRepo) ListByRun(ctx context.Context, runID uuid.UUID) ([]*agent.RunToolCall, error) {
	rows := make([]agentToolCallRow, 0)

	err := dbFromContext(ctx, r.db).
		Where("run_id = ?", runID).
		Order("created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list agent tool calls: %w", err)
	}

	calls := make([]*agent.RunToolCall, 0, len(rows))
	for i := range rows {
		calls = append(calls, rows[i].toDomain())
	}

	return calls, nil
}
