// Package chunker 定义内容切片接口。
// 参考技术方案 §12.3。
package chunker

import "github.com/beihai0xff/snowy/internal/content"

// Chunker 文档切片接口。
type Chunker interface {
	// Chunk 将文档按策略切片。
	// 切片策略：课本/讲义按段落小节，考纲按知识项，题库按题干/答案/解析，
	// 生物内容按概念/过程阶段/实验要素。
	Chunk(doc *content.Document) ([]*content.Chunk, error)
}
