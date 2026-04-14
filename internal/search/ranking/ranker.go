package ranking

import (
	"context"

	"github.com/beihai0xff/snowy/internal/search"
)

// Ranker 结果重排接口。
type Ranker interface {
	// Rank 对检索结果进行重排序。
	Rank(ctx context.Context, results []search.Result, query *search.ParsedQuery) []search.Result
}
