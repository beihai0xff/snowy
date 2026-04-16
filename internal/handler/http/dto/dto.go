// Package dto 定义 HTTP 请求/响应 DTO，与领域模型隔离。
package dto

import "github.com/google/uuid"

// ── Agent ────────────────────────────────────────────────

// ChatReq Agent 会话请求 DTO，参考技术方案 §17.1。
type ChatReq struct {
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message"              binding:"required"`
	Mode      string `json:"mode"                 binding:"omitempty,oneof=search physics biology auto"`
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
	Mode string `binding:"required,oneof=search physics biology auto" json:"mode"`
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
	Query     string `binding:"required" json:"query"`
	SessionID string `                   json:"session_id,omitempty"`
	Filters   struct {
		Subject string `json:"subject,omitempty"`
		Grade   string `json:"grade,omitempty"`
		Chapter string `json:"chapter,omitempty"`
		Source  string `json:"source,omitempty"`
	} `                   json:"filters,omitempty"`
}

// ── Physics ──────────────────────────────────────────────

// PhysicsAnalyzeReq 物理解析请求 DTO，参考技术方案 §17.3。
type PhysicsAnalyzeReq struct {
	Question string `binding:"required" json:"question"`
	Context  string `                   json:"context,omitempty"`
}

// PhysicsSimulateReq 物理调参请求 DTO，参考技术方案 §17.4。
type PhysicsSimulateReq struct {
	ModelType  string             `binding:"required" json:"model_type"`
	Parameters map[string]float64 `binding:"required" json:"parameters"`
}

// ── Biology ──────────────────────────────────────────────

// BiologyAnalyzeReq 生物解析请求 DTO，参考技术方案 §17.5。
type BiologyAnalyzeReq struct {
	Question string `binding:"required" json:"question"`
	Context  string `                   json:"context,omitempty"`
}

// ── User ─────────────────────────────────────────────────

// GoogleLoginReq Google OAuth 登录请求 — 前端用 Google ID token 换取本平台 JWT。
type GoogleLoginReq struct {
	IDToken string `binding:"required" json:"id_token"`
}

// LoginResp 登录响应。
type LoginResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// FavoriteReq 收藏请求。
type FavoriteReq struct {
	TargetType string `binding:"required,oneof=search physics biology" json:"target_type"`
	TargetID   string `binding:"required"                              json:"target_id"`
	Title      string `binding:"required"                              json:"title"`
}

// RecommendationItem 首页推荐条目。
type RecommendationItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"` // search / physics / biology
	Icon        string `json:"icon,omitempty"`
}

// RecommendationsResp 首页推荐响应。
type RecommendationsResp struct {
	HotTopics      []RecommendationItem `json:"hot_topics"`
	PhysicsModels  []RecommendationItem `json:"physics_models"`
	BiologyTopics  []RecommendationItem `json:"biology_topics"`
}
