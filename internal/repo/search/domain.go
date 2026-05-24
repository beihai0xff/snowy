// Package search 定义知识点直答域的领域模型与预留检索接口。
// 有界上下文：Knowledge Answer — 查询理解、LLM 直答、结构化学习辅助、参考信息拼装。
// 当前默认运行链路不接入外部检索仓库；OpenSearch/Embedding 作为后续可选扩展。
package search

import (
	"time"

	"github.com/google/uuid"
)

// Query 知识点回答请求领域模型。
type Query struct {
	SessionID uuid.UUID `json:"session_id,omitempty"`
	UserID    uuid.UUID `json:"user_id,omitempty"`
	Text      string    `json:"text"`
	Filters   Filters   `json:"filters,omitempty"`
}

// Filters 检索过滤条件。
type Filters struct {
	Subject string `json:"subject,omitempty"` // physics / biology / ...
	Grade   string `json:"grade,omitempty"`   // high_school
	Chapter string `json:"chapter,omitempty"`
	Source  string `json:"source,omitempty"` // textbook / exam / exercise / lecture
}

// ParsedQuery 经查询理解后的结构化查询。
type ParsedQuery struct {
	Original  string    `json:"original"`
	Keywords  []string  `json:"keywords"`
	Entities  []string  `json:"entities"`
	Intent    string    `json:"intent"`
	Embedding []float64 `json:"-"`
}

// Result 单条可选检索结果；默认 LLM 直答链路不会生成该类型。
type Result struct {
	DocID      string   `json:"doc_id"`
	SourceType string   `json:"source_type"`
	Subject    string   `json:"subject"`
	Chapter    string   `json:"chapter,omitempty"`
	Snippet    string   `json:"snippet"`
	Score      float64  `json:"score"`
	Tags       []string `json:"tags,omitempty"`
}

// Citation 表示回答参考信息；当前默认值通常为 runtime grounding。
type Citation struct {
	DocID      string  `json:"doc_id"`
	SourceType string  `json:"source_type"`
	Snippet    string  `json:"snippet"`
	Score      float64 `json:"score"`
}

// RelatedQuestion 相关问题推荐。
type RelatedQuestion struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// MisconceptionTip describes a common mistake and a corrective hint for evidence-based learning search.
type MisconceptionTip struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Correction  string `json:"correction,omitempty"`
}

// FormulaCard explains a rule/formula, its variables and applicability boundaries.
type FormulaCard struct {
	Name       string   `json:"name"`
	Expression string   `json:"expression"`
	Variables  []string `json:"variables,omitempty"`
	AppliesTo  []string `json:"applies_to,omitempty"`
	Limits     []string `json:"limits,omitempty"`
}

// ExamMapping describes how the concept appears in high-school exercise/exam scenarios.
type ExamMapping struct {
	QuestionType string   `json:"question_type"`
	Focus        string   `json:"focus"`
	PracticeHint string   `json:"practice_hint,omitempty"`
	Knowledge    []string `json:"knowledge,omitempty"`
}

// LearningAction is a suggested next step that can jump into modeling or review flows.
type LearningAction struct {
	Type        string   `json:"type"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Target      string   `json:"target,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// Response 知识点直答响应。
type Response struct {
	AnswerID         string             `json:"answer_id,omitempty"`
	Answer           string             `json:"answer"`
	KnowledgeTags    []string           `json:"knowledge_tags"`
	Citations        []Citation         `json:"citations"`
	RelatedQuestions []RelatedQuestion  `json:"related_questions"`
	Misconceptions   []MisconceptionTip `json:"misconceptions,omitempty"`
	FormulaCards     []FormulaCard      `json:"formula_cards,omitempty"`
	ExamMappings     []ExamMapping      `json:"exam_mappings,omitempty"`
	NextActions      []LearningAction   `json:"next_actions,omitempty"`
	Confidence       float64            `json:"confidence"`
}

// AnswerRecord stores the durable learning value produced by a search answer.
// It intentionally keeps evidence and tags as structured fields so later ranking,
// recommendation and audit jobs can consume them without replaying the LLM call.
type AnswerRecord struct {
	ID            uuid.UUID      `json:"id"`
	UserID        uuid.UUID      `json:"user_id"`
	SessionID     uuid.UUID      `json:"session_id,omitempty"`
	Query         string         `json:"query"`
	AnswerSummary string         `json:"answer_summary"`
	KnowledgeTags []string       `json:"knowledge_tags"`
	Citations     []Citation     `json:"citations"`
	Confidence    float64        `json:"confidence"`
	Source        string         `json:"source"`
	ModelName     string         `json:"model_name,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}
