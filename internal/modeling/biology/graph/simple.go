package graph

import (
	"fmt"

	"github.com/beihai0xff/snowy/internal/modeling/biology/domain"
)

type simpleDiagramBuilder struct{}

// NewSimpleDiagramBuilder 创建默认流程图构建器。
func NewSimpleDiagramBuilder() DiagramBuilder {
	return &simpleDiagramBuilder{}
}

func (b *simpleDiagramBuilder) Build(concepts []domain.Concept, relations []domain.Relation, title string) (*domain.DiagramSpec, error) {
	nodes := make([]domain.DiagramNode, 0, len(concepts))
	for i, concept := range concepts {
		nodes = append(nodes, domain.DiagramNode{
			ID:    fmt.Sprintf("n%d", i+1),
			Label: concept.Name,
			Type:  concept.Type,
		})
	}
	edges := make([]domain.DiagramEdge, 0, len(relations))
	for _, relation := range relations {
		edges = append(edges, domain.DiagramEdge{Source: relation.Source, Target: relation.Target, Label: relation.Type})
	}
	if title == "" {
		title = "生物概念关系图"
	}
	return &domain.DiagramSpec{DiagramType: "flow", Title: title, Nodes: nodes, Edges: edges}, nil
}
