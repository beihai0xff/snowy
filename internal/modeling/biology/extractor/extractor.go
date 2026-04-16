// Package extractor 定义生物概念抽取接口。
package extractor

import (
	"context"

	"github.com/beihai0xff/snowy/internal/modeling/biology/domain"
)

// ConceptExtractor 概念识别与关系抽取接口。
type ConceptExtractor interface {
	// Extract 从文本中抽取概念和关系。
	Extract(ctx context.Context, text string) ([]domain.Concept, []domain.Relation, error)
}
