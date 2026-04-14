package search

import "context"

// Repository 检索持久化/索引端口（DDD Port）。
// 由基础设施层（OpenSearch adapter）实现。
type Repository interface {
	// Search 执行检索（全文+向量+标签混合）。
	Search(ctx context.Context, query *ParsedQuery, filters Filters, offset, limit int) ([]Result, int64, error)
	// GetByDocID 根据文档 ID 获取详情。
	GetByDocID(ctx context.Context, docID string) (*Result, error)
}

// LogRepository 检索日志持久化端口。
type LogRepository interface {
	// SaveLog 保存检索日志。
	SaveLog(ctx context.Context, log *SearchLog) error
}

// SearchLog 检索行为日志。
type SearchLog struct {
	QueryText   string  `json:"query_text"`
	UserID      string  `json:"user_id"`
	ResultCount int     `json:"result_count"`
	LatencyMS   int     `json:"latency_ms"`
	TopScore    float64 `json:"top_score"`
}
