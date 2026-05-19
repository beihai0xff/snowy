// Package service 定义生物建模域的应用服务。
package service

import (
	"context"

	"github.com/beihai0xff/snowy/internal/modeling/biology/domain"
)

// BiologyService 生物建模域应用服务接口。
type BiologyService interface {
	// Analyze 分析生物问题，返回结构化建模结果。
	Analyze(ctx context.Context, question string, sessionContext string) (*domain.BiologyModel, error)
}

// ExperimentAnalyzer identifies experiment variables from biology questions.
type ExperimentAnalyzer interface {
	AnalyzeVariables(ctx context.Context, text string) (*domain.ExperimentVariables, error)
}

// DiagramBuilder builds renderable concept diagrams from concepts and relations.
type DiagramBuilder interface {
	Build(concepts []domain.Concept, relations []domain.Relation, title string) (*domain.DiagramSpec, error)
}

// ConceptExtractor extracts concepts and relations from text.
type ConceptExtractor interface {
	Extract(ctx context.Context, text string) ([]domain.Concept, []domain.Relation, error)
}
