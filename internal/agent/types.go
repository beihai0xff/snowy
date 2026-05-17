// Package agent 定义 Agent 编排域的核心类型与接口。
// 有界上下文：Agent Orchestration — 意图识别、工具编排、模型路由、上下文管理、结果组装。
// 参考技术方案 §10。
package agent

import (
	"time"

	"github.com/google/uuid"
)

// Mode Agent 执行模式。
type Mode string

const (
	ModeSearch    Mode = "search"
	ModePhysics   Mode = "physics"
	ModeBiology   Mode = "biology"
	ModeChemistry Mode = "chemistry" // v7 §5 预留，后端实现待 C 序列
	ModeAuto      Mode = "auto"
)

// ChatRequest Agent 会话请求，参考技术方案 §17.1。
// v7：新增追问 / 重生成上下文，参考 docs/snowy-v7-conversational-modeling-platform.md §3。
type ChatRequest struct {
	SessionID        uuid.UUID        `json:"session_id,omitempty"`
	Message          string           `json:"message"                     binding:"required"`
	Mode             Mode             `json:"mode"`
	Filters          Filters          `json:"filters,omitempty"`
	ParentPackageID  *uuid.UUID       `json:"parent_package_id,omitempty"`
	RegenerateReason string           `json:"regenerate_reason,omitempty"`
	InteractiveDemo  *DemoRequestHint `json:"interactive_demo,omitempty"`
}

// DemoRequestHint 会话化内嵌演示的可选提示，由前端透传给 demo_planner。
type DemoRequestHint struct {
	Domain     string         `json:"domain,omitempty"`
	TargetMode string         `json:"target_mode,omitempty"`
	Overrides  map[string]any `json:"overrides,omitempty"`
}

// PreviewPayload 流式 preview 事件的数据体。
// v7 §3 锁定：覆盖到最新 complete 的语义由 StreamResponseAggregator 实现。
type PreviewPayload struct {
	PackageID uuid.UUID `json:"package_id,omitempty"`
	Status    string    `json:"status"`            // partial / complete / failed
	Stage     string    `json:"stage,omitempty"`   // compile / recompute / regenerate
	Package   any       `json:"package,omitempty"` // GenerativeModelPackage 或子集
	Error     string    `json:"error,omitempty"`
}

// Filters 请求过滤条件。
type Filters struct {
	Subject string `json:"subject,omitempty"`
	Grade   string `json:"grade,omitempty"`
}

// ChatResponse Agent 会话响应，参考技术方案 §17.1。
type ChatResponse struct {
	Mode              Mode       `json:"mode"`
	Answer            string     `json:"answer"`
	Citations         []Citation `json:"citations,omitempty"`
	ToolCalls         []ToolCall `json:"tool_calls,omitempty"`
	StructuredPayload any        `json:"structured_payload,omitempty"`
	Confidence        float64    `json:"confidence"`
	NextActions       []string   `json:"next_actions,omitempty"`
}

// Citation 引用。
type Citation struct {
	DocID      string  `json:"doc_id"`
	SourceType string  `json:"source_type"`
	Snippet    string  `json:"snippet"`
	Score      float64 `json:"score"`
}

// ToolCall 工具调用记录。
type ToolCall struct {
	Tool   string `json:"tool"`
	Status string `json:"status"`
}

// ── SSE Event 类型，参考技术方案 §17.8 ────────────────────

// SSEEventType SSE 事件类型。
type SSEEventType string

const (
	SSEEventThinking   SSEEventType = "thinking"
	SSEEventContent    SSEEventType = "content"
	SSEEventCitation   SSEEventType = "citation"
	SSEEventToolCall   SSEEventType = "tool_call"
	SSEEventHeartbeat  SSEEventType = "heartbeat"
	SSEEventDiagram    SSEEventType = "diagram"
	SSEEventRenderCode SSEEventType = "render_code"
	SSEEventPreview    SSEEventType = "preview"
	SSEEventDone       SSEEventType = "done"
)

// SSEEvent 流式输出事件。
type SSEEvent struct {
	Event SSEEventType `json:"event"`
	Data  any          `json:"data"`
}

// ── Agent 会话与运行记录模型，参考技术方案 §18.2 ──────────

// Session Agent 会话。
type Session struct {
	ID        uuid.UUID      `json:"id"`
	UserID    uuid.UUID      `json:"user_id"`
	Mode      Mode           `json:"mode"`
	Status    string         `json:"status"` // active / closed
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// Message 对话消息。
// v7：PackageID nullable，关联 assistant 消息生成的 GenerativeModelPackage 快照。
type Message struct {
	ID        uuid.UUID  `json:"id"`
	SessionID uuid.UUID  `json:"session_id"`
	Role      string     `json:"role"` // user / assistant / system
	Content   string     `json:"content"`
	PackageID *uuid.UUID `json:"package_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// MessageEvent 会话回放的 SSE 事件落库记录（agent_message_events）。
// v7 §3 锁定：保留事件 seq 以支持 messages/:id/replay。
type MessageEvent struct {
	ID        uuid.UUID    `json:"id"`
	MessageID uuid.UUID    `json:"message_id"`
	Seq       int          `json:"seq"`
	Event     SSEEventType `json:"event"`
	Data      any          `json:"data,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

// Run 一次 Agent 执行记录，参考技术方案 §18.2.1。
type Run struct {
	ID             uuid.UUID `json:"id"`
	SessionID      uuid.UUID `json:"session_id"`
	MessageID      uuid.UUID `json:"message_id"`
	Mode           Mode      `json:"mode"`
	ModelName      string    `json:"model_name"`
	PromptVersion  string    `json:"prompt_version"`
	InputTokens    int       `json:"input_tokens"`
	OutputTokens   int       `json:"output_tokens"`
	EstimatedCost  float64   `json:"estimated_cost"`
	LatencyMS      int       `json:"latency_ms"`
	Confidence     float64   `json:"confidence"`
	FallbackReason string    `json:"fallback_reason,omitempty"`
	Status         string    `json:"status"` // success / failed / fallback
	ErrorCode      string    `json:"error_code,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// RunToolCall 工具调用记录，参考技术方案 §18.2.1。
type RunToolCall struct {
	ID        uuid.UUID `json:"id"`
	RunID     uuid.UUID `json:"run_id"`
	ToolName  string    `json:"tool_name"`
	Input     any       `json:"input"`
	Output    any       `json:"output"`
	LatencyMS int       `json:"latency_ms"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
