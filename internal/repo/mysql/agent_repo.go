package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/agent"
)

// agentSessionRepo 实现 agent.SessionRepository 接口。
type agentSessionRepo struct {
	db *sql.DB
}

// NewAgentSessionRepository 创建 Agent Session Repository。
func NewAgentSessionRepository(db *sql.DB) agent.SessionRepository {
	return &agentSessionRepo{db: db}
}

func (r *agentSessionRepo) Create(ctx context.Context, s *agent.Session) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO agent_sessions (id, user_id, mode, status, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.UserID, s.Mode, s.Status, jsonMap(s.Metadata), s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert agent session: %w", err)
	}
	return nil
}

func (r *agentSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*agent.Session, error) {
	s := &agent.Session{}
	var meta jsonMap
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, mode, status, metadata, created_at, updated_at
		 FROM agent_sessions WHERE id = ?`, id,
	).Scan(&s.ID, &s.UserID, &s.Mode, &s.Status, &meta, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get agent session: %w", err)
	}
	s.Metadata = map[string]any(meta)
	return s, nil
}

func (r *agentSessionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE agent_sessions SET status = ?, updated_at = NOW() WHERE id = ?`, status, id,
	)
	return err
}

func (r *agentSessionRepo) ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*agent.Session, int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM agent_sessions WHERE user_id = ?`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count agent sessions: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, mode, status, metadata, created_at, updated_at
		 FROM agent_sessions WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list agent sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*agent.Session
	for rows.Next() {
		s := &agent.Session{}
		var meta jsonMap
		if err := rows.Scan(&s.ID, &s.UserID, &s.Mode, &s.Status, &meta, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan agent session: %w", err)
		}
		s.Metadata = map[string]any(meta)
		sessions = append(sessions, s)
	}
	return sessions, total, nil
}

// agentRunRepo 实现 agent.RunRepository 接口。
type agentRunRepo struct {
	db *sql.DB
}

// NewAgentRunRepository 创建 Agent Run Repository。
func NewAgentRunRepository(db *sql.DB) agent.RunRepository {
	return &agentRunRepo{db: db}
}

func (r *agentRunRepo) Save(ctx context.Context, run *agent.Run) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO agent_runs (id, session_id, message_id, mode, model_name, prompt_version,
		 input_tokens, output_tokens, estimated_cost, latency_ms, confidence, fallback_reason,
		 status, error_code, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		run.ID, run.SessionID, run.MessageID, run.Mode, run.ModelName, run.PromptVersion,
		run.InputTokens, run.OutputTokens, run.EstimatedCost, run.LatencyMS, run.Confidence,
		run.FallbackReason, run.Status, run.ErrorCode, run.CreatedAt,
	)
	return err
}

func (r *agentRunRepo) GetByID(ctx context.Context, id uuid.UUID) (*agent.Run, error) {
	run := &agent.Run{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, session_id, message_id, mode, model_name, prompt_version,
		 input_tokens, output_tokens, estimated_cost, latency_ms, confidence,
		 fallback_reason, status, error_code, created_at
		 FROM agent_runs WHERE id = ?`, id,
	).Scan(&run.ID, &run.SessionID, &run.MessageID, &run.Mode, &run.ModelName, &run.PromptVersion,
		&run.InputTokens, &run.OutputTokens, &run.EstimatedCost, &run.LatencyMS, &run.Confidence,
		&run.FallbackReason, &run.Status, &run.ErrorCode, &run.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get agent run: %w", err)
	}
	return run, nil
}

// agentMessageRepo 实现 agent.MessageRepository 接口。
type agentMessageRepo struct {
	db *sql.DB
}

// NewAgentMessageRepository 创建 Agent Message Repository。
func NewAgentMessageRepository(db *sql.DB) agent.MessageRepository {
	return &agentMessageRepo{db: db}
}

func (r *agentMessageRepo) Save(ctx context.Context, msg *agent.Message) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO agent_messages (id, session_id, role, content, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		msg.ID, msg.SessionID, msg.Role, msg.Content, msg.CreatedAt,
	)
	return err
}

func (r *agentMessageRepo) ListBySession(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*agent.Message, int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM agent_messages WHERE session_id = ?`, sessionID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, session_id, role, content, created_at
		 FROM agent_messages WHERE session_id = ? ORDER BY created_at ASC LIMIT ? OFFSET ?`,
		sessionID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var msgs []*agent.Message
	for rows.Next() {
		m := &agent.Message{}
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, 0, err
		}
		msgs = append(msgs, m)
	}
	return msgs, total, nil
}

// agentToolCallRepo 实现 agent.ToolCallRepository 接口。
type agentToolCallRepo struct {
	db *sql.DB
}

// NewAgentToolCallRepository 创建 Agent ToolCall Repository。
func NewAgentToolCallRepository(db *sql.DB) agent.ToolCallRepository {
	return &agentToolCallRepo{db: db}
}

func (r *agentToolCallRepo) Save(ctx context.Context, tc *agent.RunToolCall) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO agent_tool_calls (id, run_id, tool_name, input, output, latency_ms, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tc.ID, tc.RunID, tc.ToolName, jsonValueOf(tc.Input), jsonValueOf(tc.Output),
		tc.LatencyMS, tc.Status, tc.CreatedAt,
	)
	return err
}

func (r *agentToolCallRepo) ListByRun(ctx context.Context, runID uuid.UUID) ([]*agent.RunToolCall, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, run_id, tool_name, input, output, latency_ms, status, created_at
		 FROM agent_tool_calls WHERE run_id = ? ORDER BY created_at ASC`, runID,
	)
	if err != nil {
		return nil, fmt.Errorf("list agent tool calls: %w", err)
	}
	defer rows.Close()

	var calls []*agent.RunToolCall
	for rows.Next() {
		tc := &agent.RunToolCall{}
		var input, output jsonMap
		if err := rows.Scan(&tc.ID, &tc.RunID, &tc.ToolName, &input, &output,
			&tc.LatencyMS, &tc.Status, &tc.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan agent tool call: %w", err)
		}
		tc.Input = map[string]any(input)
		tc.Output = map[string]any(output)
		calls = append(calls, tc)
	}
	return calls, nil
}
