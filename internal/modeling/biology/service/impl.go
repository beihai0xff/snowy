package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/beihai0xff/snowy/internal/modeling/biology/domain"
	experimentanalyzer "github.com/beihai0xff/snowy/internal/modeling/biology/experiment"
	diagrambuilder "github.com/beihai0xff/snowy/internal/modeling/biology/graph"
)

type serviceImpl struct {
	analyzer experimentanalyzer.Analyzer
	builder  diagrambuilder.DiagramBuilder
}

// NewService 创建生物建模服务。
func NewService(analyzer experimentanalyzer.Analyzer, builder diagrambuilder.DiagramBuilder) BiologyService {
	if analyzer == nil {
		analyzer = experimentanalyzer.NewSimpleAnalyzer()
	}
	if builder == nil {
		builder = diagrambuilder.NewSimpleDiagramBuilder()
	}
	return &serviceImpl{analyzer: analyzer, builder: builder}
}

func (s *serviceImpl) Analyze(ctx context.Context, question string, sessionContext string) (*domain.BiologyModel, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, fmt.Errorf("question is empty")
	}
	fullText := strings.TrimSpace(question + " " + sessionContext)
	topic := inferTopic(fullText)
	concepts := inferConcepts(fullText)
	relations := inferRelations(concepts)
	variables, err := s.analyzer.AnalyzeVariables(ctx, fullText)
	if err != nil {
		return nil, fmt.Errorf("analyze experiment variables: %w", err)
	}
	diagram, err := s.builder.Build(concepts, relations, topic+" 关系图")
	if err != nil {
		return nil, fmt.Errorf("build biology diagram: %w", err)
	}

	return &domain.BiologyModel{
		Topic:               topic,
		Concepts:            concepts,
		Relations:           relations,
		ProcessSteps:        processSteps(topic),
		ExperimentVariables: variables,
		Diagram:             diagram,
		ResultSummary:       fmt.Sprintf("已识别主题 %s，并抽取 %d 个概念、%d 条关系。", topic, len(concepts), len(relations)),
	}, nil
}

func inferTopic(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "光合作用") || strings.Contains(lower, "photosynthesis"):
		return "photosynthesis"
	case strings.Contains(lower, "呼吸作用") || strings.Contains(lower, "respiration"):
		return "cellular_respiration"
	case strings.Contains(lower, "酶") || strings.Contains(lower, "enzyme"):
		return "enzyme_activity"
	default:
		return "biology_analysis"
	}
}

func inferConcepts(text string) []domain.Concept {
	concepts := []domain.Concept{}
	lower := strings.ToLower(text)
	add := func(name, conceptType string) {
		for _, concept := range concepts {
			if concept.Name == name {
				return
			}
		}
		concepts = append(concepts, domain.Concept{Name: name, Type: conceptType})
	}

	if strings.Contains(lower, "光") || strings.Contains(lower, "photosynthesis") {
		add("光照强度", "factor")
		add("叶绿体", "structure")
		add("有机物积累", "result")
	}
	if strings.Contains(lower, "二氧化碳") {
		add("二氧化碳浓度", "factor")
	}
	if strings.Contains(lower, "酶") {
		add("酶活性", "result")
		add("温度", "factor")
	}
	if len(concepts) == 0 {
		add("核心概念", "process")
		add("分析对象", "result")
	}

	return concepts
}

func inferRelations(concepts []domain.Concept) []domain.Relation {
	relations := make([]domain.Relation, 0, maxInt(len(concepts)-1, 0))
	for i := 0; i < len(concepts)-1; i++ {
		relations = append(relations, domain.Relation{Source: concepts[i].Name, Target: concepts[i+1].Name, Type: "influences"})
	}
	return relations
}

func processSteps(topic string) []domain.ProcessStep {
	switch topic {
	case "photosynthesis":
		return []domain.ProcessStep{
			{Index: 1, Title: "识别限制因素", Content: "判断光照、二氧化碳等因素是否构成限制条件。"},
			{Index: 2, Title: "分析物质变化", Content: "结合光反应和暗反应分析有机物积累变化。"},
		}
	case "enzyme_activity":
		return []domain.ProcessStep{
			{Index: 1, Title: "锁定变量", Content: "先区分自变量、因变量和控制变量。"},
			{Index: 2, Title: "分析趋势", Content: "结合酶活性曲线判断促进或抑制作用。"},
		}
	default:
		return []domain.ProcessStep{
			{Index: 1, Title: "提取核心概念", Content: "从题干中识别关键生物概念与过程。"},
			{Index: 2, Title: "建立关系", Content: "将概念组织为因果或流程关系。"},
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
