package search

import "context"

// Service 知识检索域应用服务接口。
type Service interface {
	// Query 执行知识检索，返回结构化响应。
	Query(ctx context.Context, q *Query) (*Response, error)
}
