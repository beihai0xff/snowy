// Package parser 定义内容解析接口。
package parser

import "github.com/beihai0xff/snowy/internal/content"

// Parser 内容解析接口。
type Parser interface {
	// Parse 解析原始内容为结构化文档。
	Parse(raw []byte, sourceType string) (*content.Document, error)
	// SupportedTypes 返回支持的内容类型。
	SupportedTypes() []string
}
