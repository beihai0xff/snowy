// Package opensearch 提供 OpenSearch 搜索引擎适配实现。
// 参考技术方案 §6.2.2。
package opensearch

import (
	"context"
	"fmt"

	internalsearch "github.com/beihai0xff/snowy/internal/repo/search"

	"github.com/beihai0xff/snowy/internal/content"
	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// OpenSearchAdapter OpenSearch 搜索适配器。
// 实现 search.Repository 和 content/indexer.Indexer 接口。
type OpenSearchAdapter struct {
	cfg config.OpenSearchConfig
	// client *opensearch.Client — 生产环境注入 OpenSearch 客户端
}

// NewOpenSearchAdapter 创建 OpenSearch 适配器。
func NewOpenSearchAdapter(cfg config.OpenSearchConfig) *OpenSearchAdapter {
	return &OpenSearchAdapter{cfg: cfg}
}

// Search 执行混合检索（全文+向量+标签）。
func (a *OpenSearchAdapter) Search(
	ctx context.Context,
	query *internalsearch.ParsedQuery,
	filters internalsearch.Filters,
	offset, limit int,
) ([]internalsearch.Result, int64, error) {
	// TODO: 构建 OpenSearch DSL 查询
	return nil, 0, fmt.Errorf("opensearch search: not implemented")
}

// GetByDocID 根据文档 ID 获取详情。
func (a *OpenSearchAdapter) GetByDocID(ctx context.Context, docID string) (*internalsearch.Result, error) {
	// TODO: OpenSearch get by ID
	return nil, fmt.Errorf("opensearch get by doc id: not implemented")
}

// Index 将切片写入 OpenSearch 索引。
func (a *OpenSearchAdapter) Index(ctx context.Context, chunks []*content.Chunk) error {
	// TODO: OpenSearch bulk index
	return fmt.Errorf("opensearch index: not implemented")
}

// Delete 从索引中删除文档切片。
func (a *OpenSearchAdapter) Delete(ctx context.Context, documentID string) error {
	// TODO: OpenSearch delete by query
	return fmt.Errorf("opensearch delete: not implemented")
}
