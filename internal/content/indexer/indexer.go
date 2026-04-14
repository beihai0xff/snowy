// Package indexer 定义内容索引接口。
// 参考技术方案 §12.4。
package indexer

import (
	"context"

	"github.com/beihai0xff/snowy/internal/content"
)

// Indexer 内容索引接口。
type Indexer interface {
	// Index 将切片写入搜索引擎索引（全文+向量+标签）。
	Index(ctx context.Context, chunks []*content.Chunk) error
	// Delete 从索引中删除文档的全部切片。
	Delete(ctx context.Context, documentID string) error
}
