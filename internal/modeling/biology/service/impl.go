//nolint:goconst // Biology scenario identifiers are clearer inline beside their rule branches.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/beihai0xff/snowy/internal/modeling/biology/domain"
	physicsdomain "github.com/beihai0xff/snowy/internal/modeling/physics/domain"
)

type serviceImpl struct {
	analyzer ExperimentAnalyzer
	builder  DiagramBuilder
}

// NewService 创建生物建模服务。
func NewService(analyzer ExperimentAnalyzer, builder DiagramBuilder) BiologyService {
	if analyzer == nil {
		analyzer = NewSimpleAnalyzer()
	}

	if builder == nil {
		builder = NewSimpleDiagramBuilder()
	}

	return &serviceImpl{analyzer: analyzer, builder: builder}
}

func (s *serviceImpl) Analyze(
	ctx context.Context,
	question string,
	sessionContext string,
) (*domain.BiologyModel, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, errors.New("question is empty")
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

	steps := processSteps(topic)

	return &domain.BiologyModel{
		Topic:               topic,
		Concepts:            concepts,
		Relations:           relations,
		ProcessSteps:        steps,
		ExperimentVariables: variables,
		Diagram:             diagram,
		SceneSpec:           buildBiologySceneSpec(topic, concepts, relations, steps, variables),
		ResultSummary:       fmt.Sprintf("已识别主题 %s，并抽取 %d 个概念、%d 条关系。", topic, len(concepts), len(relations)),
	}, nil
}

func buildBiologySceneSpec(
	topic string,
	concepts []domain.Concept,
	relations []domain.Relation,
	steps []domain.ProcessStep,
	variables *domain.ExperimentVariables,
) *physicsdomain.SceneSpec {
	sceneType := "biology_concept_flow"
	title := "生物概念动态演示"
	summary := "将概念、关系和过程步骤转化为浏览器中的粒子流、阶段面板和动态关系网络。"

	switch topic {
	case "photosynthesis":
		sceneType = "biology_photosynthesis_3d"
		title = "光合作用 3D 动态演示"
		summary = "用叶绿体舞台、光子粒子和 CO₂/H₂O/O₂/糖分子流动展示光合作用过程。"
	case "cellular_respiration", "enzyme_activity":
		sceneType = "biology_cell_process_3d"
		title = "细胞过程 3D 可视化演示"
		summary = "用细胞膜、细胞器、变量标签和物质运输路径展示细胞层面的动态机制。"
	}

	props := map[string]float64{
		"concept_count":   float64(len(concepts)),
		"relation_count":  float64(len(relations)),
		"step_count":      float64(len(steps)),
		"animation_speed": 1,
	}
	if variables != nil {
		props["independent_count"] = float64(len(variables.Independent))
		props["dependent_count"] = float64(len(variables.Dependent))
		props["controlled_count"] = float64(len(variables.Controlled))
	}

	return &physicsdomain.SceneSpec{
		SceneType:    sceneType,
		Title:        title,
		Summary:      summary + " 概念：" + conceptNames(concepts) + "。过程：" + stepTitles(steps) + "。",
		RenderMode:   physicsdomain.RenderModeHTMLIframe,
		DefaultProps: props,
	}
}

func conceptNames(concepts []domain.Concept) string {
	if len(concepts) == 0 {
		return "核心概念"
	}

	names := make([]string, 0, len(concepts))
	for _, concept := range concepts {
		if strings.TrimSpace(concept.Name) != "" {
			names = append(names, concept.Name)
		}
	}

	if len(names) == 0 {
		return "核心概念"
	}

	return strings.Join(names, "、")
}

func stepTitles(steps []domain.ProcessStep) string {
	if len(steps) == 0 {
		return "概念提取、关系建立"
	}

	titles := make([]string, 0, len(steps))
	for _, step := range steps {
		if strings.TrimSpace(step.Title) != "" {
			titles = append(titles, step.Title)
		}
	}

	if len(titles) == 0 {
		return "概念提取、关系建立"
	}

	return strings.Join(titles, " → ")
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
	for i := range len(concepts) - 1 {
		relations = append(
			relations,
			domain.Relation{Source: concepts[i].Name, Target: concepts[i+1].Name, Type: "influences"},
		)
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
