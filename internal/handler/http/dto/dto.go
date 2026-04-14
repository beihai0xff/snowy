// Package dto 定义 HTTP 请求/响应 DTO，与领域模型隔离。
package dto

import "github.com/google/uuid"

// ── Agent ────────────────────────────────────────────────

// ChatReq Agent 会话请求 DTO，参考技术方案 §17.1。
type ChatReq struct {
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message" binding:"required"`
	Mode      string `json:"mode" binding:"omitempty,oneof=search physics biology auto"`
	Filters   struct {
		Subject string `json:"subject,omitempty"`
		Grade   string `json:"grade,omitempty"`
	} `json:"filters,omitempty"`
}

// ChatResp Agent 会话响应 DTO。
type ChatResp struct {
	Mode              string   `json:"mode"`
	Answer            string   `json:"answer"`
	Citations         []any    `json:"citations,omitempty"`
	ToolCalls         []any    `json:"tool_calls,omitempty"`
	StructuredPayload any      `json:"structured_payload,omitempty"`
	Confidence        float64  `json:"confidence"`
	NextActions       []string `json:"next_actions,omitempty"`
}

// CreateSessionReq 创建会话请求。
type CreateSessionReq struct {
	Mode string `json:"mode" binding:"required,oneof=search physics biology auto"`
}

// SessionResp 会话响应。
type SessionResp struct {
	ID        uuid.UUID `json:"id"`
	Mode      string    `json:"mode"`
	Status    string    `json:"status"`
	CreatedAt string    `json:"created_at"`
}

// ── Search ───────────────────────────────────────────────

// SearchQueryReq 搜索请求 DTO，参考技术方案 §17.2。
type SearchQueryReq struct {
	Query     string `json:"query" binding:"required"`
	SessionID string `json:"session_id,omitempty"`
	Filters   struct {
		Subject string `json:"subject,omitempty"`
		Grade   string `json:"grade,omitempty"`
		Chapter string `json:"chapter,omitempty"`
		Source  string `json:"source,omitempty"`
	} `json:"filters,omitempty"`
}

// ── Physics ──────────────────────────────────────────────

// PhysicsAnalyzeReq 物理解析请求 DTO，参考技术方案 §17.3。
type PhysicsAnalyzeReq struct {
	Question string `json:"question" binding:"required"`
	Context  string `json:"context,omitempty"`
}

// PhysicsSimulateReq 物理调参请求 DTO，参考技术方案 §17.4。
type PhysicsSimulateReq struct {
	ModelType  string             `json:"model_type" binding:"required"`
	Parameters map[string]float64 `json:"parameters" binding:"required"`
}

// ── Biology ──────────────────────────────────────────────

// BiologyAnalyzeReq 生物解析请求 DTO，参考技术方案 §17.5。
type BiologyAnalyzeReq struct {
	Question string `json:"question" binding:"required"`
	Context  string `json:"context,omitempty"`
}

// ── User ─────────────────────────────────────────────────

// LoginReq 登录请求。
type LoginReq struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

// LoginResp 登录响应。
type LoginResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RegisterReq 注册请求。
type RegisterReq struct {
	Phone    string `json:"phone" binding:"required"`
	Nickname string `json:"nickname" binding:"required"`
}

// FavoriteReq 收藏请求。
type FavoriteReq struct {
	TargetType string `json:"target_type" binding:"required,oneof=search physics biology"`
	TargetID   string `json:"target_id" binding:"required"`
	Title      string `json:"title" binding:"required"`
}
