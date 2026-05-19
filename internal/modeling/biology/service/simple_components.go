package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/beihai0xff/snowy/internal/modeling/biology/domain"
)

type simpleAnalyzer struct{}

func NewSimpleAnalyzer() ExperimentAnalyzer {
	return &simpleAnalyzer{}
}

func (a *simpleAnalyzer) AnalyzeVariables(_ context.Context, text string) (*domain.ExperimentVariables, error) {
	lower := strings.ToLower(text)
	vars := &domain.ExperimentVariables{}

	switch {
	case strings.Contains(lower, "光") || strings.Contains(lower, "photosynthesis"):
		vars.Independent = []string{"光照强度"}
		vars.Dependent = []string{"有机物积累"}
		vars.Controlled = []string{"温度", "二氧化碳浓度"}
	case strings.Contains(lower, "酶"):
		vars.Independent = []string{"温度"}
		vars.Dependent = []string{"酶活性"}
		vars.Controlled = []string{"pH", "底物浓度"}
	default:
		vars.Controlled = []string{"需补充实验条件"}
	}

	return vars, nil
}

type simpleDiagramBuilder struct{}

func NewSimpleDiagramBuilder() DiagramBuilder {
	return &simpleDiagramBuilder{}
}

func (b *simpleDiagramBuilder) Build(
	concepts []domain.Concept,
	relations []domain.Relation,
	title string,
) (*domain.DiagramSpec, error) {
	nodes := make([]domain.DiagramNode, 0, len(concepts))

	nameToID := make(map[string]string, len(concepts))
	for i, concept := range concepts {
		id := fmt.Sprintf("n%d", i+1)
		nodes = append(nodes, domain.DiagramNode{
			ID:    id,
			Label: concept.Name,
			Type:  concept.Type,
		})
		nameToID[concept.Name] = id
	}

	edges := make([]domain.DiagramEdge, 0, len(relations))
	for _, relation := range relations {
		sourceID, sourceOK := nameToID[relation.Source]

		targetID, targetOK := nameToID[relation.Target]
		if !sourceOK || !targetOK || sourceID == "" || targetID == "" {
			continue
		}

		edges = append(edges, domain.DiagramEdge{
			Source: sourceID,
			Target: targetID,
			Label:  relation.Type,
		})
	}

	if len(nodes) == 0 {
		nodes = append(nodes, domain.DiagramNode{ID: "n1", Label: "分析对象", Type: "process"})
	}

	if title == "" {
		title = "生物概念关系图"
	}

	return &domain.DiagramSpec{DiagramType: "flow", Title: title, Nodes: nodes, Edges: edges}, nil
}
