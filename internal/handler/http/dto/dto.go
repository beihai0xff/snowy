// Package dto 定义 HTTP 请求/响应 DTO，与领域模型隔离。
package dto

import "github.com/google/uuid"

// ── Agent ────────────────────────────────────────────────

// ChatReq Agent 会话请求 DTO，参考技术方案 §17.1。
// v7 §3：新增 parent_package_id / regenerate_reason / interactive_demo。
type ChatReq struct {
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message"                     binding:"required"`
	Mode      string `json:"mode"                        binding:"omitempty,oneof=search physics biology chemistry auto"`
	Filters   struct {
		Subject string `json:"subject,omitempty"`
		Grade   string `json:"grade,omitempty"`
	} `json:"filters,omitempty"`
	ParentPackageID  string              `json:"parent_package_id,omitempty"`
	RegenerateReason string              `json:"regenerate_reason,omitempty"`
	InteractiveDemo  *DemoRequestHintDTO `json:"interactive_demo,omitempty"`
}

// DemoRequestHintDTO 前端透传给 demo_planner 的可选提示。
type DemoRequestHintDTO struct {
	Domain     string         `json:"domain,omitempty"`
	TargetMode string         `json:"target_mode,omitempty"`
	Overrides  map[string]any `json:"overrides,omitempty"`
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
	Question    string `binding:"required" json:"question"`
	Context     string `                   json:"context,omitempty"`
	TargetScene string `                   json:"target_scene,omitempty"`
}

// PhysicsSimulateReq 物理调参请求 DTO，参考技术方案 §17.4（兼容旧链路）。
type PhysicsSimulateReq struct {
	ModelType  string             `binding:"required" json:"model_type"`
	Parameters map[string]float64 `binding:"required" json:"parameters"`
}

// RenderSceneSpec 前端渲染场景规格。
type RenderSceneSpec struct {
	SceneType    string             `binding:"required" json:"scene_type"`
	Title        string             `                   json:"title,omitempty"`
	Summary      string             `                   json:"summary,omitempty"`
	RenderMode   string             `                   json:"render_mode,omitempty"`
	DefaultProps map[string]float64 `                   json:"default_props,omitempty"`
}

// RenderGenerateReq 前端渲染代码生成请求 DTO。
type RenderGenerateReq struct {
	SceneSpec  RenderSceneSpec `binding:"required"                                 json:"scene_spec"`
	Context    string          `                                                   json:"context,omitempty"`
	RenderMode string          `binding:"omitempty,oneof=html_iframe react_iframe" json:"render_mode,omitempty"`
}

// ── Biology ──────────────────────────────────────────────

// BiologyAnalyzeReq 生物解析请求 DTO，参考技术方案 §17.5。
type BiologyAnalyzeReq struct {
	Question string `binding:"required" json:"question"`
	Context  string `                   json:"context,omitempty"`
}

// ── Chemistry ─────────────────────────────────────────────

// ChemistryAnalyzeReq 化学反应解析请求 DTO，参考技术方案 §5。
type ChemistryAnalyzeReq struct {
	Equation string `binding:"required" json:"equation"`
	Context  string `                   json:"context,omitempty"`
}

// ChemistryBalanceReq 化学方程式配平请求 DTO。
type ChemistryBalanceReq struct {
	Equation string `binding:"required" json:"equation"`
}

// ── Share ─────────────────────────────────────────────────

// PackageShareCreateReq v7 §6.2 静态分享请求 DTO。
type PackageShareCreateReq struct {
	PackageID      string `binding:"required"                         json:"package_id"`
	Mode           string `binding:"omitempty,oneof=view interactive" json:"mode,omitempty"`
	ExpiresInHours int    `binding:"omitempty,min=0,max=8760"         json:"expires_in_hours,omitempty"`
}

// ── Auth / User ───────────────────────────────────────────

type EmailRegisterReq struct {
	Email    string `binding:"required,email" json:"email"`
	Password string `binding:"required,min=8" json:"password"`
	Nickname string `                         json:"nickname,omitempty"`
}

type EmailLoginReq struct {
	Email    string `binding:"required,email" json:"email"`
	Password string `binding:"required"       json:"password"`
}

type AuthResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         any    `json:"user"`
}

// FavoriteReq 收藏请求。
type FavoriteReq struct {
	TargetType   string         `binding:"required,oneof=search answer evidence physics biology model_package render_code model_config" json:"target_type"`
	TargetID     string         `binding:"required"                                                                                     json:"target_id"`
	Title        string         `binding:"required"                                                                                     json:"title"`
	MetadataJSON map[string]any `                                                                                                       json:"metadata_json,omitempty"`
}

type ReactionReq struct {
	TargetType   string `binding:"required,oneof=search answer evidence physics biology model_package render_code" json:"target_type"`
	TargetID     string `binding:"required"                                                                        json:"target_id"`
	ReactionType string `binding:"required,oneof=like dislike"                                                     json:"reaction_type"`
	Visibility   string `binding:"omitempty,oneof=public private"                                                  json:"visibility,omitempty"`
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
	HotTopics     []RecommendationItem `json:"hot_topics"`
	PhysicsModels []RecommendationItem `json:"physics_models"`
	BiologyTopics []RecommendationItem `json:"biology_topics"`
}

// ── Generative Modeling v4 ───────────────────────────────

type EvidenceRefDTO struct {
	DocID         string   `json:"doc_id"`
	SourceType    string   `json:"source_type"`
	Title         string   `json:"title,omitempty"`
	Chapter       string   `json:"chapter,omitempty"`
	Snippet       string   `json:"snippet"`
	KnowledgeTags []string `json:"knowledge_tags,omitempty"`
	Confidence    float64  `json:"confidence"`
}

type ModelingCompileContextReq struct {
	Citations     []EvidenceRefDTO `json:"citations,omitempty"`
	KnowledgeTags []string         `json:"knowledge_tags,omitempty"`
	SourcePage    string           `json:"source_page,omitempty"`
	UserNotes     string           `json:"user_notes,omitempty"`
}

type ModelingCompileReq struct {
	SessionID  string                    `json:"session_id,omitempty"`
	Message    string                    `json:"message"               binding:"required"`
	Domain     string                    `json:"domain,omitempty"      binding:"omitempty,oneof=auto physics biology"`
	GradeBand  string                    `json:"grade_band,omitempty"`
	TargetMode string                    `json:"target_mode,omitempty" binding:"omitempty,oneof=interactive_model review explain"`
	Context    ModelingCompileContextReq `json:"context,omitempty"`
}

// ModelingRecomputeReq v7 §3：基于父 package 与 overrides 生成新快照。
type ModelingRecomputeReq struct {
	Overrides map[string]float64 `json:"overrides"`
}

// ModelingRegenerateReq v7 §3：以父 package 为上下文重新调用 LLM。
type ModelingRegenerateReq struct {
	Reason  string                    `json:"reason"`
	Context ModelingCompileContextReq `json:"context,omitempty"`
}
