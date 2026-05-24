package search

import "context"

// QueryParser 查询理解接口。
type QueryParser interface {
	Parse(raw string) (*ParsedQuery, error)
}

// ResultRanker 结果重排接口。
type ResultRanker interface {
	Rank(ctx context.Context, results []Result, query *ParsedQuery) []Result
}

// Service 知识点直答域应用服务接口。
type Service interface {
	// Query 执行知识点直答，返回结构化学习响应。
	Query(ctx context.Context, q *Query) (*Response, error)
}
