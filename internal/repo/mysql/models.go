package mysql

import (
	"time"

	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/agent"
	"github.com/beihai0xff/snowy/internal/modeling/generative"
	"github.com/beihai0xff/snowy/internal/monitoring"
	searchdomain "github.com/beihai0xff/snowy/internal/repo/search"
	"github.com/beihai0xff/snowy/internal/user"
)

type llmCallRecordRow struct {
	ID            string    `gorm:"column:id"`
	UserID        string    `gorm:"column:user_id"`
	Role          string    `gorm:"column:role"`
	Provider      string    `gorm:"column:provider"`
	ModelProvider string    `gorm:"column:model_provider"`
	Model         string    `gorm:"column:model"`
	BaseURL       string    `gorm:"column:base_url"`
	Operation     string    `gorm:"column:operation"`
	Status        string    `gorm:"column:status"`
	LatencyMS     int64     `gorm:"column:latency_ms"`
	InputTokens   int       `gorm:"column:input_tokens"`
	OutputTokens  int       `gorm:"column:output_tokens"`
	MaxTokens     int       `gorm:"column:max_tokens"`
	Temperature   float64   `gorm:"column:temperature"`
	PromptChars   int       `gorm:"column:prompt_chars"`
	SystemPE      string    `gorm:"column:system_pe"`
	UserPrompt    string    `gorm:"column:user_prompt"`
	PromptPreview string    `gorm:"column:prompt_preview"`
	FinishReason  string    `gorm:"column:finish_reason"`
	Error         string    `gorm:"column:error"`
	StartedAt     time.Time `gorm:"column:started_at"`
	FinishedAt    time.Time `gorm:"column:finished_at"`
}

func (llmCallRecordRow) TableName() string { return "llm_call_records" }

func newLLMCallRecordRow(r monitoring.LLMCallRecord) *llmCallRecordRow {
	return &llmCallRecordRow{
		ID:            r.ID,
		UserID:        r.UserID,
		Role:          r.Role,
		Provider:      r.Provider,
		ModelProvider: r.ModelProvider,
		Model:         r.Model,
		BaseURL:       r.BaseURL,
		Operation:     r.Operation,
		Status:        r.Status,
		LatencyMS:     r.LatencyMS,
		InputTokens:   r.InputTokens,
		OutputTokens:  r.OutputTokens,
		MaxTokens:     r.MaxTokens,
		Temperature:   r.Temperature,
		PromptChars:   r.PromptChars,
		SystemPE:      r.SystemPE,
		UserPrompt:    r.UserPrompt,
		PromptPreview: r.PromptPreview,
		FinishReason:  r.FinishReason,
		Error:         r.Error,
		StartedAt:     r.StartedAt,
		FinishedAt:    r.FinishedAt,
	}
}

func (r *llmCallRecordRow) toMonitoring() monitoring.LLMCallRecord {
	if r == nil {
		return monitoring.LLMCallRecord{}
	}

	return monitoring.LLMCallRecord{
		ID:            r.ID,
		UserID:        r.UserID,
		Role:          r.Role,
		Provider:      r.Provider,
		ModelProvider: r.ModelProvider,
		Model:         r.Model,
		BaseURL:       r.BaseURL,
		Operation:     r.Operation,
		Status:        r.Status,
		LatencyMS:     r.LatencyMS,
		InputTokens:   r.InputTokens,
		OutputTokens:  r.OutputTokens,
		MaxTokens:     r.MaxTokens,
		Temperature:   r.Temperature,
		PromptChars:   r.PromptChars,
		SystemPE:      r.SystemPE,
		UserPrompt:    r.UserPrompt,
		PromptPreview: r.PromptPreview,
		FinishReason:  r.FinishReason,
		Error:         r.Error,
		StartedAt:     r.StartedAt,
		FinishedAt:    r.FinishedAt,
	}
}

//nolint:recvcheck // GORM row types intentionally mix value and pointer receivers for table metadata and conversions.
type userRow struct {
	ID           uuid.UUID `gorm:"column:id"`
	GoogleID     string    `gorm:"column:google_id"`
	Email        string    `gorm:"column:email"`
	PasswordHash string    `gorm:"column:password_hash"`
	Phone        string    `gorm:"column:phone"`
	Nickname     string    `gorm:"column:nickname"`
	Role         user.Role `gorm:"column:role"`
	AvatarURL    string    `gorm:"column:avatar_url"`
	LastLoginAt  time.Time `gorm:"column:last_login_at"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (userRow) TableName() string { return "users" }

func newUserRow(u *user.User) *userRow {
	if u == nil {
		return nil
	}

	return &userRow{
		ID:           u.ID,
		GoogleID:     u.GoogleID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Phone:        u.Phone,
		Nickname:     u.Nickname,
		Role:         u.Role,
		AvatarURL:    u.AvatarURL,
		LastLoginAt:  u.LastLoginAt,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func (r *userRow) toDomain() *user.User {
	if r == nil {
		return nil
	}

	return &user.User{
		ID:           r.ID,
		GoogleID:     r.GoogleID,
		Email:        r.Email,
		PasswordHash: r.PasswordHash,
		Phone:        r.Phone,
		Nickname:     r.Nickname,
		Role:         r.Role,
		AvatarURL:    r.AvatarURL,
		LastLoginAt:  r.LastLoginAt,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

//nolint:recvcheck // GORM row types intentionally mix value and pointer receivers for table metadata and conversions.
type favoriteRow struct {
	ID           uuid.UUID `gorm:"column:id"`
	UserID       uuid.UUID `gorm:"column:user_id"`
	TargetType   string    `gorm:"column:target_type"`
	TargetID     string    `gorm:"column:target_id"`
	Title        string    `gorm:"column:title"`
	MetadataJSON jsonMap   `gorm:"column:metadata_json"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (favoriteRow) TableName() string { return "favorites" }

func newFavoriteRow(f *user.Favorite) *favoriteRow {
	if f == nil {
		return nil
	}

	return &favoriteRow{
		ID:           f.ID,
		UserID:       f.UserID,
		TargetType:   f.TargetType,
		TargetID:     f.TargetID,
		Title:        f.Title,
		MetadataJSON: newJSONMap(f.MetadataJSON),
		CreatedAt:    f.CreatedAt,
	}
}

func (r *favoriteRow) toDomain() *user.Favorite {
	if r == nil {
		return nil
	}

	return &user.Favorite{
		ID:           r.ID,
		UserID:       r.UserID,
		TargetType:   r.TargetType,
		TargetID:     r.TargetID,
		Title:        r.Title,
		MetadataJSON: map[string]any(r.MetadataJSON),
		CreatedAt:    r.CreatedAt,
	}
}

//nolint:recvcheck // GORM row types intentionally mix value and pointer receivers for table metadata and conversions.
type answerRecordRow struct {
	ID            uuid.UUID  `gorm:"column:id"`
	UserID        uuid.UUID  `gorm:"column:user_id"`
	SessionID     *uuid.UUID `gorm:"column:session_id"`
	Query         string     `gorm:"column:query"`
	AnswerSummary string     `gorm:"column:answer_summary"`
	KnowledgeTags jsonValue  `gorm:"column:knowledge_tags"`
	Citations     jsonValue  `gorm:"column:citations"`
	Confidence    float64    `gorm:"column:confidence"`
	Source        string     `gorm:"column:source"`
	ModelName     string     `gorm:"column:model_name"`
	Metadata      jsonMap    `gorm:"column:metadata"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
}

func (answerRecordRow) TableName() string { return "answer_records" }

func newAnswerRecordRow(record *searchdomain.AnswerRecord) *answerRecordRow {
	if record == nil {
		return nil
	}

	return &answerRecordRow{
		ID:            record.ID,
		UserID:        record.UserID,
		SessionID:     nullableUUID(record.SessionID),
		Query:         record.Query,
		AnswerSummary: record.AnswerSummary,
		KnowledgeTags: newJSONValue(record.KnowledgeTags),
		Citations:     newJSONValue(record.Citations),
		Confidence:    record.Confidence,
		Source:        record.Source,
		ModelName:     record.ModelName,
		Metadata:      newJSONMap(record.Metadata),
		CreatedAt:     record.CreatedAt,
	}
}

func (r *answerRecordRow) toDomain() *searchdomain.AnswerRecord {
	if r == nil {
		return nil
	}

	record := &searchdomain.AnswerRecord{
		ID:            r.ID,
		UserID:        r.UserID,
		Query:         r.Query,
		AnswerSummary: r.AnswerSummary,
		Confidence:    r.Confidence,
		Source:        r.Source,
		ModelName:     r.ModelName,
		Metadata:      map[string]any(r.Metadata),
		CreatedAt:     r.CreatedAt,
	}
	if r.SessionID != nil {
		record.SessionID = *r.SessionID
	}
	_ = r.KnowledgeTags.AssignTo(&record.KnowledgeTags)
	_ = r.Citations.AssignTo(&record.Citations)

	return record
}

//nolint:recvcheck // GORM row types intentionally mix value and pointer receivers for table metadata and conversions.
type reactionRow struct {
	ID           uuid.UUID `gorm:"column:id"`
	UserID       uuid.UUID `gorm:"column:user_id"`
	TargetType   string    `gorm:"column:target_type"`
	TargetID     string    `gorm:"column:target_id"`
	ReactionType string    `gorm:"column:reaction_type"`
	Visibility   string    `gorm:"column:visibility"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (reactionRow) TableName() string { return "reactions" }

func newReactionRow(r *user.Reaction) *reactionRow {
	if r == nil {
		return nil
	}

	return &reactionRow{
		ID:           r.ID,
		UserID:       r.UserID,
		TargetType:   r.TargetType,
		TargetID:     r.TargetID,
		ReactionType: r.ReactionType,
		Visibility:   r.Visibility,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func (r *reactionRow) toDomain() *user.Reaction {
	if r == nil {
		return nil
	}

	return &user.Reaction{
		ID:           r.ID,
		UserID:       r.UserID,
		TargetType:   r.TargetType,
		TargetID:     r.TargetID,
		ReactionType: r.ReactionType,
		Visibility:   r.Visibility,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
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

//nolint:recvcheck // GORM row types intentionally mix value and pointer receivers for table metadata and conversions.
type generativeModelPackageRow struct {
	ID             uuid.UUID  `gorm:"column:id"`
	UserID         uuid.UUID  `gorm:"column:user_id"`
	SessionID      *uuid.UUID `gorm:"column:session_id"`
	Domain         string     `gorm:"column:domain"`
	Question       string     `gorm:"column:question"`
	PackageJSON    jsonValue  `gorm:"column:package_json"`
	ModelName      string     `gorm:"column:model_name"`
	Status         string     `gorm:"column:status"`
	Confidence     float64    `gorm:"column:confidence"`
	ValidationJSON jsonValue  `gorm:"column:validation_json"`
	FallbackReason string     `gorm:"column:fallback_reason"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
}

func (generativeModelPackageRow) TableName() string { return "generative_model_packages" }

func newGenerativeModelPackageRow(pkg *generative.GenerativeModelPackage) *generativeModelPackageRow {
	if pkg == nil {
		return nil
	}

	return &generativeModelPackageRow{
		ID:             pkg.PackageID,
		UserID:         pkg.UserID,
		SessionID:      nullableUUID(pkg.SessionID),
		Domain:         pkg.Domain,
		Question:       pkg.Question,
		PackageJSON:    newJSONValue(pkg),
		ModelName:      pkg.ModelName,
		Status:         pkg.Status,
		Confidence:     pkg.Confidence,
		ValidationJSON: newJSONValue(pkg.ValidationReport),
		FallbackReason: pkg.FallbackReason,
		CreatedAt:      pkg.CreatedAt,
	}
}

func (r *generativeModelPackageRow) toDomain() *generative.GenerativeModelPackage {
	if r == nil {
		return nil
	}

	var pkg generative.GenerativeModelPackage

	_ = r.PackageJSON.AssignTo(&pkg)
	if pkg.PackageID == uuid.Nil {
		pkg.PackageID = r.ID
	}

	pkg.UserID = r.UserID
	if r.SessionID != nil {
		pkg.SessionID = *r.SessionID
	}

	pkg.Domain = firstNonEmptyString(pkg.Domain, r.Domain)
	pkg.Question = firstNonEmptyString(pkg.Question, r.Question)
	pkg.ModelName = firstNonEmptyString(pkg.ModelName, r.ModelName)

	pkg.Status = firstNonEmptyString(pkg.Status, r.Status)
	if pkg.Confidence == 0 {
		pkg.Confidence = r.Confidence
	}

	pkg.FallbackReason = firstNonEmptyString(pkg.FallbackReason, r.FallbackReason)
	if pkg.CreatedAt.IsZero() {
		pkg.CreatedAt = r.CreatedAt
	}

	return &pkg
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

//nolint:recvcheck // GORM row types intentionally mix value and pointer receivers for table metadata and conversions.
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

//nolint:recvcheck // GORM row types intentionally mix value and pointer receivers for table metadata and conversions.
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

//nolint:recvcheck // GORM row types intentionally mix value and pointer receivers for table metadata and conversions.
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

//nolint:recvcheck // GORM row types intentionally mix value and pointer receivers for table metadata and conversions.
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
