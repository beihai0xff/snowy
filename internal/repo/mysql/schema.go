//nolint:lll // GORM index tags are intentionally kept with their schema fields.
package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const mysqlTableOptions = "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci"

type llmCallRecordSchema struct {
	ID            string    `gorm:"column:id;type:char(36);primaryKey"`
	UserID        string    `gorm:"column:user_id;type:char(36);not null;default:'';index:idx_llm_calls_user,priority:1"`
	Role          string    `gorm:"column:role;type:varchar(32);not null;default:''"`
	Provider      string    `gorm:"column:provider;type:varchar(64);not null;default:'';index:idx_llm_calls_provider,priority:1"`
	ModelProvider string    `gorm:"column:model_provider;type:varchar(64);not null;default:''"`
	Model         string    `gorm:"column:model;type:varchar(128);not null;default:'';index:idx_llm_calls_model,priority:1"`
	BaseURL       string    `gorm:"column:base_url;type:text;not null"`
	Operation     string    `gorm:"column:operation;type:varchar(64);not null;default:'';index:idx_llm_calls_operation,priority:1"`
	Status        string    `gorm:"column:status;type:varchar(16);not null;default:''"`
	LatencyMS     int64     `gorm:"column:latency_ms;not null;default:0"`
	InputTokens   int       `gorm:"column:input_tokens;not null;default:0"`
	OutputTokens  int       `gorm:"column:output_tokens;not null;default:0"`
	MaxTokens     int       `gorm:"column:max_tokens;not null;default:0"`
	Temperature   float64   `gorm:"column:temperature;type:decimal(6,4);not null;default:0"`
	PromptChars   int       `gorm:"column:prompt_chars;not null;default:0"`
	SystemPE      string    `gorm:"column:system_pe;type:mediumtext"`
	UserPrompt    string    `gorm:"column:user_prompt;type:mediumtext"`
	PromptPreview string    `gorm:"column:prompt_preview;type:mediumtext"`
	FinishReason  string    `gorm:"column:finish_reason;type:varchar(64);not null;default:''"`
	Error         string    `gorm:"column:error;type:text"`
	StartedAt     time.Time `gorm:"column:started_at;type:datetime(3);not null"`
	FinishedAt    time.Time `gorm:"column:finished_at;type:datetime(3);not null;index:idx_llm_calls_user,priority:2,sort:desc;index:idx_llm_calls_provider,priority:2,sort:desc;index:idx_llm_calls_model,priority:2,sort:desc;index:idx_llm_calls_operation,priority:2,sort:desc;index:idx_llm_calls_finished,sort:desc"`
}

func (llmCallRecordSchema) TableName() string { return "llm_call_records" }

type userSchema struct {
	ID           uuid.UUID `gorm:"column:id;type:char(36);primaryKey"`
	GoogleID     string    `gorm:"column:google_id;type:varchar(128);not null;index:idx_users_google_id;default:''"`
	Email        string    `gorm:"column:email;type:varchar(255);not null;default:'';index:idx_users_email"`
	PasswordHash string    `gorm:"column:password_hash;type:varchar(255);not null;default:''"`
	Phone        string    `gorm:"column:phone;type:varchar(20);not null;default:''"`
	Nickname     string    `gorm:"column:nickname;type:varchar(64);not null;default:''"`
	Role         string    `gorm:"column:role;type:varchar(16);not null;default:'student'"`
	AvatarURL    string    `gorm:"column:avatar_url;type:text;not null"`
	LastLoginAt  time.Time `gorm:"column:last_login_at;type:datetime(3);not null"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime(3);not null"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:datetime(3);not null"`
}

func (userSchema) TableName() string { return "users" }

type favoriteSchema struct {
	ID           uuid.UUID `gorm:"column:id;type:char(36);primaryKey"`
	UserID       uuid.UUID `gorm:"column:user_id;type:char(36);not null;index:idx_favorites_user,priority:1"`
	TargetType   string    `gorm:"column:target_type;type:varchar(32);not null"`
	TargetID     string    `gorm:"column:target_id;type:varchar(128);not null"`
	Title        string    `gorm:"column:title;type:text;not null"`
	MetadataJSON jsonMap   `gorm:"column:metadata_json;type:json"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime(3);not null;index:idx_favorites_user,priority:2,sort:desc"`
}

func (favoriteSchema) TableName() string { return "favorites" }

type answerRecordSchema struct {
	ID            uuid.UUID  `gorm:"column:id;type:char(36);primaryKey"`
	UserID        uuid.UUID  `gorm:"column:user_id;type:char(36);not null;index:idx_answer_records_user,priority:1"`
	SessionID     *uuid.UUID `gorm:"column:session_id;type:char(36);index:idx_answer_records_session"`
	Query         string     `gorm:"column:query;type:text;not null"`
	AnswerSummary string     `gorm:"column:answer_summary;type:mediumtext;not null"`
	KnowledgeTags jsonValue  `gorm:"column:knowledge_tags;type:json;not null"`
	Citations     jsonValue  `gorm:"column:citations;type:json;not null"`
	Confidence    float64    `gorm:"column:confidence;type:decimal(5,4);not null;default:0"`
	Source        string     `gorm:"column:source;type:varchar(32);not null;default:'';index:idx_answer_records_source,priority:1"`
	ModelName     string     `gorm:"column:model_name;type:varchar(128);not null;default:''"`
	Metadata      jsonMap    `gorm:"column:metadata;type:json"`
	CreatedAt     time.Time  `gorm:"column:created_at;type:datetime(3);not null;index:idx_answer_records_user,priority:2,sort:desc;index:idx_answer_records_source,priority:2,sort:desc"`
}

func (answerRecordSchema) TableName() string { return "answer_records" }

type reactionSchema struct {
	ID           uuid.UUID `gorm:"column:id;type:char(36);primaryKey"`
	UserID       uuid.UUID `gorm:"column:user_id;type:char(36);not null;uniqueIndex:uk_reactions_user_target,priority:1;index:idx_reactions_user,priority:1;index:idx_reactions_target,priority:3"`
	TargetType   string    `gorm:"column:target_type;type:varchar(32);not null;uniqueIndex:uk_reactions_user_target,priority:2;index:idx_reactions_target,priority:1"`
	TargetID     string    `gorm:"column:target_id;type:varchar(128);not null;uniqueIndex:uk_reactions_user_target,priority:3;index:idx_reactions_target,priority:2"`
	ReactionType string    `gorm:"column:reaction_type;type:varchar(16);not null"`
	Visibility   string    `gorm:"column:visibility;type:varchar(16);not null;default:'public'"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime(3);not null;index:idx_reactions_user,priority:2,sort:desc"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:datetime(3);not null"`
}

func (reactionSchema) TableName() string { return "reactions" }

type agentSessionSchema struct {
	ID        uuid.UUID `gorm:"column:id;type:char(36);primaryKey"`
	UserID    uuid.UUID `gorm:"column:user_id;type:char(36);not null;index:idx_agent_sessions_user,priority:1"`
	Mode      string    `gorm:"column:mode;type:varchar(32);not null;default:'auto'"`
	Status    string    `gorm:"column:status;type:varchar(16);not null;default:'active'"`
	Metadata  jsonMap   `gorm:"column:metadata;type:json"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime(3);not null;index:idx_agent_sessions_user,priority:2,sort:desc"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime(3);not null"`
}

func (agentSessionSchema) TableName() string { return "agent_sessions" }

type agentMessageSchema struct {
	ID        uuid.UUID  `gorm:"column:id;type:char(36);primaryKey"`
	SessionID uuid.UUID  `gorm:"column:session_id;type:char(36);not null;index:idx_agent_messages_session,priority:1"`
	Role      string     `gorm:"column:role;type:varchar(16);not null"`
	Content   string     `gorm:"column:content;type:text;not null"`
	PackageID *uuid.UUID `gorm:"column:package_id;type:char(36);index:idx_agent_messages_package"`
	CreatedAt time.Time  `gorm:"column:created_at;type:datetime(3);not null;index:idx_agent_messages_session,priority:2,sort:asc"`
}

func (agentMessageSchema) TableName() string { return "agent_messages" }

// agentMessageEventSchema v7 §3：SSE 事件按 seq 落库以支持 messages/:id/replay。
type agentMessageEventSchema struct {
	ID        uuid.UUID `gorm:"column:id;type:char(36);primaryKey"`
	MessageID uuid.UUID `gorm:"column:message_id;type:char(36);not null;index:idx_agent_message_events_message,priority:1"`
	Seq       int       `gorm:"column:seq;not null;default:0;index:idx_agent_message_events_message,priority:2,sort:asc"`
	Event     string    `gorm:"column:event;type:varchar(32);not null"`
	Data      jsonValue `gorm:"column:data;type:json"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime(3);not null"`
}

func (agentMessageEventSchema) TableName() string { return "agent_message_events" }

type agentRunSchema struct {
	ID             uuid.UUID `gorm:"column:id;type:char(36);primaryKey"`
	SessionID      uuid.UUID `gorm:"column:session_id;type:char(36);not null;index:idx_agent_runs_session,priority:1"`
	MessageID      uuid.UUID `gorm:"column:message_id;type:char(36);not null"`
	Mode           string    `gorm:"column:mode;type:varchar(32);not null"`
	ModelName      string    `gorm:"column:model_name;type:varchar(64);not null"`
	PromptVersion  string    `gorm:"column:prompt_version;type:varchar(32);not null;default:''"`
	InputTokens    int       `gorm:"column:input_tokens;not null;default:0"`
	OutputTokens   int       `gorm:"column:output_tokens;not null;default:0"`
	EstimatedCost  float64   `gorm:"column:estimated_cost;type:decimal(10,6);not null;default:0"`
	LatencyMS      int       `gorm:"column:latency_ms;not null;default:0"`
	Confidence     float64   `gorm:"column:confidence;type:decimal(4,3);not null;default:0"`
	FallbackReason string    `gorm:"column:fallback_reason;type:varchar(128);default:''"`
	Status         string    `gorm:"column:status;type:varchar(16);not null;default:'success'"`
	ErrorCode      string    `gorm:"column:error_code;type:varchar(64);default:''"`
	CreatedAt      time.Time `gorm:"column:created_at;type:datetime(3);not null;index:idx_agent_runs_session,priority:2,sort:desc"`
}

func (agentRunSchema) TableName() string { return "agent_runs" }

type agentToolCallSchema struct {
	ID        uuid.UUID `gorm:"column:id;type:char(36);primaryKey"`
	RunID     uuid.UUID `gorm:"column:run_id;type:char(36);not null;index:idx_agent_tool_calls_run"`
	ToolName  string    `gorm:"column:tool_name;type:varchar(64);not null"`
	Input     jsonValue `gorm:"column:input;type:json"`
	Output    jsonValue `gorm:"column:output;type:json"`
	LatencyMS int       `gorm:"column:latency_ms;not null;default:0"`
	Status    string    `gorm:"column:status;type:varchar(16);not null;default:'success'"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime(3);not null"`
}

func (agentToolCallSchema) TableName() string { return "agent_tool_calls" }

type generativeModelPackageSchema struct {
	ID             uuid.UUID  `gorm:"column:id;type:char(36);primaryKey"`
	UserID         uuid.UUID  `gorm:"column:user_id;type:char(36);not null;index:idx_generative_packages_user,priority:1"`
	SessionID      *uuid.UUID `gorm:"column:session_id;type:char(36);index:idx_generative_packages_session"`
	Domain         string     `gorm:"column:domain;type:varchar(32);not null;index:idx_generative_packages_domain,priority:1"`
	Question       string     `gorm:"column:question;type:text;not null"`
	PackageJSON    jsonValue  `gorm:"column:package_json;type:json;not null"`
	ModelName      string     `gorm:"column:model_name;type:varchar(128);not null;default:''"`
	Status         string     `gorm:"column:status;type:varchar(32);not null;default:'success'"`
	Confidence     float64    `gorm:"column:confidence;type:decimal(5,4);not null;default:0"`
	ValidationJSON jsonValue  `gorm:"column:validation_json;type:json"`
	FallbackReason string     `gorm:"column:fallback_reason;type:varchar(255);default:''"`
	CreatedAt      time.Time  `gorm:"column:created_at;type:datetime(3);not null;index:idx_generative_packages_user,priority:2,sort:desc;index:idx_generative_packages_domain,priority:2,sort:desc"`
}

func (generativeModelPackageSchema) TableName() string { return "generative_model_packages" }

type contentDocumentSchema struct {
	ID              uuid.UUID `gorm:"column:id;type:char(36);primaryKey"`
	DocID           string    `gorm:"column:doc_id;type:varchar(128);not null;uniqueIndex:uk_content_docs_doc_id"`
	SourceType      string    `gorm:"column:source_type;type:varchar(32);not null;index:idx_content_docs_source"`
	Subject         string    `gorm:"column:subject;type:varchar(32);not null;index:idx_content_docs_subject"`
	Grade           string    `gorm:"column:grade;type:varchar(32);not null;default:''"`
	Chapter         string    `gorm:"column:chapter;type:varchar(128);not null;default:''"`
	Section         string    `gorm:"column:section;type:varchar(128);not null;default:''"`
	KnowledgeTags   jsonValue `gorm:"column:knowledge_tags;type:json;not null"`
	TopicTags       jsonValue `gorm:"column:topic_tags;type:json;not null"`
	Difficulty      string    `gorm:"column:difficulty;type:varchar(16);default:''"`
	Content         string    `gorm:"column:content;type:longtext;not null"`
	Answer          *string   `gorm:"column:answer;type:longtext"`
	Metadata        jsonValue `gorm:"column:metadata;type:json"`
	CopyrightStatus string    `gorm:"column:copyright_status;type:varchar(32);not null;default:'unknown'"`
	Version         int       `gorm:"column:version;not null;default:1"`
	CreatedAt       time.Time `gorm:"column:created_at;type:datetime(3);not null"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:datetime(3);not null"`
}

func (contentDocumentSchema) TableName() string { return "content_documents" }

type contentChunkSchema struct {
	ID         uuid.UUID `gorm:"column:id;type:char(36);primaryKey"`
	DocumentID uuid.UUID `gorm:"column:document_id;type:char(36);not null;index:idx_content_chunks_doc,priority:1"`
	ChunkIndex int       `gorm:"column:chunk_index;not null;index:idx_content_chunks_doc,priority:2"`
	Content    string    `gorm:"column:content;type:longtext;not null"`
	Tags       jsonValue `gorm:"column:tags;type:json;not null"`
	ChunkType  string    `gorm:"column:chunk_type;type:varchar(32);not null;default:'paragraph'"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime(3);not null"`
}

func (contentChunkSchema) TableName() string { return "content_chunks" }

type searchLogSchema struct {
	ID          string    `gorm:"column:id;type:char(36);primaryKey;default:(UUID())"`
	QueryText   string    `gorm:"column:query_text;type:text;not null"`
	UserID      string    `gorm:"column:user_id;type:varchar(64);default:''"`
	ResultCount int       `gorm:"column:result_count;not null;default:0"`
	LatencyMS   int       `gorm:"column:latency_ms;not null;default:0"`
	TopScore    float64   `gorm:"column:top_score;type:decimal(5,4);not null;default:0"`
	CreatedAt   time.Time `gorm:"column:created_at;type:datetime(3);not null"`
}

func (searchLogSchema) TableName() string { return "search_logs" }

type physicsRunSchema struct {
	ID             uuid.UUID `gorm:"column:id;type:char(36);primaryKey"`
	SessionID      uuid.UUID `gorm:"column:session_id;type:char(36);not null"`
	Question       string    `gorm:"column:question;type:text;not null"`
	Result         jsonValue `gorm:"column:result;type:json"`
	ModelName      string    `gorm:"column:model_name;type:varchar(64);not null;default:''"`
	LatencyMS      int       `gorm:"column:latency_ms;not null;default:0"`
	Status         string    `gorm:"column:status;type:varchar(16);not null;default:'success'"`
	FallbackReason string    `gorm:"column:fallback_reason;type:varchar(128);default:''"`
	CreatedAt      time.Time `gorm:"column:created_at;type:datetime(3);not null"`
}

func (physicsRunSchema) TableName() string { return "physics_runs" }

type biologyRunSchema struct {
	ID             uuid.UUID `gorm:"column:id;type:char(36);primaryKey"`
	SessionID      uuid.UUID `gorm:"column:session_id;type:char(36);not null"`
	Question       string    `gorm:"column:question;type:text;not null"`
	Result         jsonValue `gorm:"column:result;type:json"`
	ModelName      string    `gorm:"column:model_name;type:varchar(64);not null;default:''"`
	LatencyMS      int       `gorm:"column:latency_ms;not null;default:0"`
	Status         string    `gorm:"column:status;type:varchar(16);not null;default:'success'"`
	FallbackReason string    `gorm:"column:fallback_reason;type:varchar(128);default:''"`
	CreatedAt      time.Time `gorm:"column:created_at;type:datetime(3);not null"`
}

func (biologyRunSchema) TableName() string { return "biology_runs" }

type conceptGraphSnapshotSchema struct {
	ID           string     `gorm:"column:id;type:char(36);primaryKey;default:(UUID())"`
	BiologyRunID *uuid.UUID `gorm:"column:biology_run_id;type:char(36)"`
	Diagram      jsonValue  `gorm:"column:diagram;type:json;not null"`
	CreatedAt    time.Time  `gorm:"column:created_at;type:datetime(3);not null"`
}

func (conceptGraphSnapshotSchema) TableName() string { return "concept_graph_snapshots" }

type historySchema struct {
	ID         uuid.UUID  `gorm:"column:id;type:char(36);primaryKey"`
	UserID     uuid.UUID  `gorm:"column:user_id;type:char(36);not null;index:idx_history_items_user,priority:1"`
	ActionType string     `gorm:"column:action_type;type:varchar(32);not null"`
	Query      string     `gorm:"column:query;type:text;not null"`
	SessionID  *uuid.UUID `gorm:"column:session_id;type:char(36)"`
	CreatedAt  time.Time  `gorm:"column:created_at;type:datetime(3);not null;index:idx_history_items_user,priority:2,sort:desc"`
}

func (historySchema) TableName() string { return "history_items" }

type promptTemplateSchema struct {
	ID         string    `gorm:"column:id;type:char(36);primaryKey;default:(UUID())"`
	Scene      string    `gorm:"column:scene;type:varchar(64);not null;uniqueIndex:uk_scene_version,priority:1"`
	Version    string    `gorm:"column:version;type:varchar(32);not null;uniqueIndex:uk_scene_version,priority:2"`
	Template   string    `gorm:"column:template;type:longtext;not null"`
	Status     string    `gorm:"column:status;type:varchar(16);not null;default:'active'"`
	TrafficPct int       `gorm:"column:traffic_pct;not null;default:100"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime(3);not null"`
}

func (promptTemplateSchema) TableName() string { return "prompt_templates" }

// packageShareSchema v7 §6.2：静态分享包。
type packageShareSchema struct {
	Token        string     `gorm:"column:token;type:char(32);primaryKey"`
	PackageID    uuid.UUID  `gorm:"column:package_id;type:char(36);not null;index:idx_package_shares_pkg"`
	SnapshotJSON jsonValue  `gorm:"column:snapshot_json;type:longtext;not null"`
	CreatedBy    uuid.UUID  `gorm:"column:created_by;type:char(36);not null;index:idx_package_shares_user,priority:1"`
	ExpiresAt    *time.Time `gorm:"column:expires_at;type:datetime(3)"`
	Mode         string     `gorm:"column:mode;type:varchar(32);not null;default:'view'"`
	CreatedAt    time.Time  `gorm:"column:created_at;type:datetime(3);not null;index:idx_package_shares_user,priority:2,sort:desc"`
}

func (packageShareSchema) TableName() string { return "package_shares" }

func schemaModels() []any {
	return []any{
		&llmCallRecordSchema{},
		&userSchema{},
		&favoriteSchema{},
		&answerRecordSchema{},
		&reactionSchema{},
		&agentSessionSchema{},
		&agentMessageSchema{},
		&agentMessageEventSchema{},
		&agentRunSchema{},
		&agentToolCallSchema{},
		&generativeModelPackageSchema{},
		&contentDocumentSchema{},
		&contentChunkSchema{},
		&searchLogSchema{},
		&physicsRunSchema{},
		&biologyRunSchema{},
		&conceptGraphSnapshotSchema{},
		&historySchema{},
		&promptTemplateSchema{},
		&packageShareSchema{},
	}
}

// RunMigrations 使用 GORM 统一初始化 / 演进 MySQL Schema。
func RunMigrations(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("migration db is nil")
	}

	gdb := db.WithContext(ctx)
	for _, model := range schemaModels() {
		if err := gdb.Set("gorm:table_options", mysqlTableOptions).AutoMigrate(model); err != nil {
			return fmt.Errorf("auto migrate %T: %w", model, err)
		}
	}

	return nil
}
