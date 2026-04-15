package mysql

import (
	"time"

	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/agent"
	"github.com/beihai0xff/snowy/internal/user"
)

type userRow struct {
	ID          uuid.UUID `gorm:"column:id"`
	Phone       string    `gorm:"column:phone"`
	Nickname    string    `gorm:"column:nickname"`
	Role        user.Role `gorm:"column:role"`
	AvatarURL   string    `gorm:"column:avatar_url"`
	LastLoginAt time.Time `gorm:"column:last_login_at"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (userRow) TableName() string { return "users" }

func newUserRow(u *user.User) *userRow {
	if u == nil {
		return nil
	}

	return &userRow{
		ID:          u.ID,
		Phone:       u.Phone,
		Nickname:    u.Nickname,
		Role:        u.Role,
		AvatarURL:   u.AvatarURL,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func (r *userRow) toDomain() *user.User {
	if r == nil {
		return nil
	}

	return &user.User{
		ID:          r.ID,
		Phone:       r.Phone,
		Nickname:    r.Nickname,
		Role:        r.Role,
		AvatarURL:   r.AvatarURL,
		LastLoginAt: r.LastLoginAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

type favoriteRow struct {
	ID         uuid.UUID `gorm:"column:id"`
	UserID     uuid.UUID `gorm:"column:user_id"`
	TargetType string    `gorm:"column:target_type"`
	TargetID   string    `gorm:"column:target_id"`
	Title      string    `gorm:"column:title"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (favoriteRow) TableName() string { return "favorites" }

func newFavoriteRow(f *user.Favorite) *favoriteRow {
	if f == nil {
		return nil
	}

	return &favoriteRow{
		ID:         f.ID,
		UserID:     f.UserID,
		TargetType: f.TargetType,
		TargetID:   f.TargetID,
		Title:      f.Title,
		CreatedAt:  f.CreatedAt,
	}
}

func (r *favoriteRow) toDomain() *user.Favorite {
	if r == nil {
		return nil
	}

	return &user.Favorite{
		ID:         r.ID,
		UserID:     r.UserID,
		TargetType: r.TargetType,
		TargetID:   r.TargetID,
		Title:      r.Title,
		CreatedAt:  r.CreatedAt,
	}
}

type historyRow struct {
	ID         uuid.UUID  `gorm:"column:id"`
	UserID     uuid.UUID  `gorm:"column:user_id"`
	ActionType string     `gorm:"column:action_type"`
	Query      string     `gorm:"column:query"`
	SessionID  *uuid.UUID `gorm:"column:session_id"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
}

func (historyRow) TableName() string { return "history_items" }

func newHistoryRow(item *user.HistoryItem) *historyRow {
	if item == nil {
		return nil
	}

	return &historyRow{
		ID:         item.ID,
		UserID:     item.UserID,
		ActionType: item.ActionType,
		Query:      item.Query,
		SessionID:  nullableUUID(item.SessionID),
		CreatedAt:  item.CreatedAt,
	}
}

func (r *historyRow) toDomain() *user.HistoryItem {
	if r == nil {
		return nil
	}

	item := &user.HistoryItem{
		ID:         r.ID,
		UserID:     r.UserID,
		ActionType: r.ActionType,
		Query:      r.Query,
		CreatedAt:  r.CreatedAt,
	}
	if r.SessionID != nil {
		item.SessionID = *r.SessionID
	}

	return item
}

type agentSessionRow struct {
	ID        uuid.UUID  `gorm:"column:id"`
	UserID    uuid.UUID  `gorm:"column:user_id"`
	Mode      agent.Mode `gorm:"column:mode"`
	Status    string     `gorm:"column:status"`
	Metadata  jsonMap    `gorm:"column:metadata"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

func (agentSessionRow) TableName() string { return "agent_sessions" }

func newAgentSessionRow(s *agent.Session) *agentSessionRow {
	if s == nil {
		return nil
	}

	return &agentSessionRow{
		ID:        s.ID,
		UserID:    s.UserID,
		Mode:      s.Mode,
		Status:    s.Status,
		Metadata:  newJSONMap(s.Metadata),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func (r *agentSessionRow) toDomain() *agent.Session {
	if r == nil {
		return nil
	}

	return &agent.Session{
		ID:        r.ID,
		UserID:    r.UserID,
		Mode:      r.Mode,
		Status:    r.Status,
		Metadata:  map[string]any(r.Metadata),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

type agentMessageRow struct {
	ID        uuid.UUID `gorm:"column:id"`
	SessionID uuid.UUID `gorm:"column:session_id"`
	Role      string    `gorm:"column:role"`
	Content   string    `gorm:"column:content"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (agentMessageRow) TableName() string { return "agent_messages" }

func newAgentMessageRow(msg *agent.Message) *agentMessageRow {
	if msg == nil {
		return nil
	}

	return &agentMessageRow{
		ID:        msg.ID,
		SessionID: msg.SessionID,
		Role:      msg.Role,
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}
}

func (r *agentMessageRow) toDomain() *agent.Message {
	if r == nil {
		return nil
	}

	return &agent.Message{
		ID:        r.ID,
		SessionID: r.SessionID,
		Role:      r.Role,
		Content:   r.Content,
		CreatedAt: r.CreatedAt,
	}
}

type agentRunRow struct {
	ID             uuid.UUID  `gorm:"column:id"`
	SessionID      uuid.UUID  `gorm:"column:session_id"`
	MessageID      uuid.UUID  `gorm:"column:message_id"`
	Mode           agent.Mode `gorm:"column:mode"`
	ModelName      string     `gorm:"column:model_name"`
	PromptVersion  string     `gorm:"column:prompt_version"`
	InputTokens    int        `gorm:"column:input_tokens"`
	OutputTokens   int        `gorm:"column:output_tokens"`
	EstimatedCost  float64    `gorm:"column:estimated_cost"`
	LatencyMS      int        `gorm:"column:latency_ms"`
	Confidence     float64    `gorm:"column:confidence"`
	FallbackReason string     `gorm:"column:fallback_reason"`
	Status         string     `gorm:"column:status"`
	ErrorCode      string     `gorm:"column:error_code"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
}

func (agentRunRow) TableName() string { return "agent_runs" }

func newAgentRunRow(run *agent.Run) *agentRunRow {
	if run == nil {
		return nil
	}

	return &agentRunRow{
		ID:             run.ID,
		SessionID:      run.SessionID,
		MessageID:      run.MessageID,
		Mode:           run.Mode,
		ModelName:      run.ModelName,
		PromptVersion:  run.PromptVersion,
		InputTokens:    run.InputTokens,
		OutputTokens:   run.OutputTokens,
		EstimatedCost:  run.EstimatedCost,
		LatencyMS:      run.LatencyMS,
		Confidence:     run.Confidence,
		FallbackReason: run.FallbackReason,
		Status:         run.Status,
		ErrorCode:      run.ErrorCode,
		CreatedAt:      run.CreatedAt,
	}
}

func (r *agentRunRow) toDomain() *agent.Run {
	if r == nil {
		return nil
	}

	return &agent.Run{
		ID:             r.ID,
		SessionID:      r.SessionID,
		MessageID:      r.MessageID,
		Mode:           r.Mode,
		ModelName:      r.ModelName,
		PromptVersion:  r.PromptVersion,
		InputTokens:    r.InputTokens,
		OutputTokens:   r.OutputTokens,
		EstimatedCost:  r.EstimatedCost,
		LatencyMS:      r.LatencyMS,
		Confidence:     r.Confidence,
		FallbackReason: r.FallbackReason,
		Status:         r.Status,
		ErrorCode:      r.ErrorCode,
		CreatedAt:      r.CreatedAt,
	}
}

type agentToolCallRow struct {
	ID        uuid.UUID `gorm:"column:id"`
	RunID     uuid.UUID `gorm:"column:run_id"`
	ToolName  string    `gorm:"column:tool_name"`
	Input     jsonValue `gorm:"column:input"`
	Output    jsonValue `gorm:"column:output"`
	LatencyMS int       `gorm:"column:latency_ms"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (agentToolCallRow) TableName() string { return "agent_tool_calls" }

func newAgentToolCallRow(tc *agent.RunToolCall) *agentToolCallRow {
	if tc == nil {
		return nil
	}

	return &agentToolCallRow{
		ID:        tc.ID,
		RunID:     tc.RunID,
		ToolName:  tc.ToolName,
		Input:     newJSONValue(tc.Input),
		Output:    newJSONValue(tc.Output),
		LatencyMS: tc.LatencyMS,
		Status:    tc.Status,
		CreatedAt: tc.CreatedAt,
	}
}

func (r *agentToolCallRow) toDomain() *agent.RunToolCall {
	if r == nil {
		return nil
	}

	return &agent.RunToolCall{
		ID:        r.ID,
		RunID:     r.RunID,
		ToolName:  r.ToolName,
		Input:     r.Input.Data,
		Output:    r.Output.Data,
		LatencyMS: r.LatencyMS,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
	}
}

func nullableUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}

	v := id

	return &v
}
