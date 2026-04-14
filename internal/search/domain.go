// Package search 定义知识检索域的领域模型与接口。
// 有界上下文：Knowledge Search — 查询理解、多路召回、结果重排、引用拼装。
// 参考技术方案 §9.3 & §11.1。
package search

import "github.com/google/uuid"

// Query 检索请求领域模型。
type Query struct {
	SessionID uuid.UUID `json:"session_id,omitempty"`
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

// Result 单条检索结果。
type Result struct {
	DocID      string   `json:"doc_id"`
	SourceType string   `json:"source_type"`
	Subject    string   `json:"subject"`
	Chapter    string   `json:"chapter,omitempty"`
	Snippet    string   `json:"snippet"`
	Score      float64  `json:"score"`
	Tags       []string `json:"tags,omitempty"`
}

// Citation 引用片段，参考技术方案 §13.3。
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

// Response 检索响应，参考技术方案 §13.3。
type Response struct {
	Answer           string            `json:"answer"`
	KnowledgeTags    []string          `json:"knowledge_tags"`
	Citations        []Citation        `json:"citations"`
	RelatedQuestions []RelatedQuestion `json:"related_questions"`
	Confidence       float64           `json:"confidence"`
}
