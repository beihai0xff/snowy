package agent

import (
	"context"

	"github.com/google/uuid"
)

// Service Agent 编排域应用服务接口。
type Service interface {
	// Chat 同步模式：处理一次会话请求并返回完整结果。
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	// ChatStream 流式模式：处理请求并通过 channel 逐步推送 SSE 事件。
	ChatStream(ctx context.Context, req *ChatRequest, events chan<- SSEEvent) error
}

// SessionRepository Agent 会话持久化端口。
type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	GetByID(ctx context.Context, id uuid.UUID) (*Session, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Session, int64, error)
}

// MessageRepository 消息持久化端口。
type MessageRepository interface {
	Save(ctx context.Context, msg *Message) error
	ListBySession(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*Message, int64, error)
}

// RunRepository Agent 运行记录持久化端口。
type RunRepository interface {
	Save(ctx context.Context, run *Run) error
	GetByID(ctx context.Context, id uuid.UUID) (*Run, error)
}

// ToolCallRepository 工具调用记录持久化端口。
type ToolCallRepository interface {
	Save(ctx context.Context, tc *RunToolCall) error
	ListByRun(ctx context.Context, runID uuid.UUID) ([]*RunToolCall, error)
}
