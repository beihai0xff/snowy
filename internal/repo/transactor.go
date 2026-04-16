package repo

import "context"

// Transactor 定义跨 repository 的事务执行端口。
// 具体实现由基础设施层（如 mysql）提供。
type Transactor interface {
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}
