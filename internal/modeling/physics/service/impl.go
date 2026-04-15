package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/beihai0xff/snowy/internal/modeling/physics/calculator"
	"github.com/beihai0xff/snowy/internal/modeling/physics/domain"
)

var numericConditionPattern = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s*([a-zA-Z°/%μ]*)`)

type serviceImpl struct {
	calculator calculator.Calculator
}

// NewService 创建物理建模服务。
func NewService(calc calculator.Calculator) PhysicsService {
	if calc == nil {
		calc = calculator.NewSimpleCalculator()
	}
	return &serviceImpl{calculator: calc}
}

func (s *serviceImpl) Analyze(_ context.Context, question string, sessionContext string) (*domain.PhysicsModel, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, fmt.Errorf("question is empty")
	}

	modelType := inferModelType(question + " " + sessionContext)
	conditions, params := extractConditions(question)
	if len(conditions) == 0 {
		params = defaultParameters(modelType)
	}

	computeResult, err := s.calculator.Compute(modelType, params)
	if err != nil {
		computeResult = &domain.ComputeResult{Warnings: []string{err.Error()}}
	}

	warnings := append([]string(nil), computeResult.Warnings...)
	if len(conditions) == 0 {
		warnings = append(warnings, "题干未抽取到明确数值，结果基于默认参数模板")
	}

	return &domain.PhysicsModel{
		ModelType:     modelType,
		Conditions:    conditions,
		Steps:         derivationSteps(modelType),
		ResultSummary: resultSummary(modelType, computeResult.Values),
		Warnings:      warnings,
		Chart:         computeResult.Chart,
		Parameters:    parameterSchema(modelType),
	}, nil
}

func (s *serviceImpl) Simulate(_ context.Context, modelType domain.ModelType, params map[string]float64) (*domain.ComputeResult, error) {
	return s.calculator.Compute(modelType, params)
}

func inferModelType(text string) domain.ModelType {
	lower := strings.ToLower(text)
	for _, rule := range []struct {
		model    domain.ModelType
		keywords []string
	}{
		{model: domain.ModelProjectileMotion, keywords: []string{"平抛", "抛", "projectile"}},
		{model: domain.ModelNewtonSecondLaw, keywords: []string{"牛顿", "force", "受力"}},
		{model: domain.ModelUniformAcceleration, keywords: []string{"加速度", "acceler", "匀变速"}},
		{model: domain.ModelWorkEnergy, keywords: []string{"功", "energy", "能量"}},
		{model: domain.ModelSpringOscillator, keywords: []string{"弹簧", "oscillat", "简谐"}},
		{model: domain.ModelTwoBodyMotion, keywords: []string{"双体", "引力", "orbit", "轨道"}},
	} {
		if containsAny(lower, rule.keywords...) {
			return rule.model
		}
	}

	return domain.ModelUniformMotion
}

func extractConditions(question string) ([]domain.Condition, map[string]float64) {
	matches := numericConditionPattern.FindAllStringSubmatch(question, -1)
	conditions := make([]domain.Condition, 0, len(matches))
	params := map[string]float64{}
	keys := []string{"v0", "angle_deg", "t", "a", "m"}
	for i, match := range matches {
		if len(match) < 2 {
			continue
		}
		value, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			continue
		}
		name := fmt.Sprintf("value_%d", i+1)
		if i < len(keys) {
			name = keys[i]
			params[name] = value
		}
		conditions = append(conditions, domain.Condition{Name: name, Value: value, Unit: strings.TrimSpace(match[2])})
	}

	return conditions, params
}

func derivationSteps(modelType domain.ModelType) []domain.DerivationStep {
	switch modelType {
	case domain.ModelProjectileMotion:
		return []domain.DerivationStep{
			{Index: 1, Title: "分解初速度", Content: "将初速度分解为水平和竖直两个方向。"},
			{Index: 2, Title: "列位移方程", Content: "水平做匀速运动，竖直做匀变速运动。"},
		}
	case domain.ModelNewtonSecondLaw:
		return []domain.DerivationStep{
			{Index: 1, Title: "识别受力", Content: "明确研究对象和合外力。"},
			{Index: 2, Title: "应用牛顿第二定律", Content: "建立 F = ma 的数量关系。"},
		}
	case domain.ModelUniformAcceleration:
		return []domain.DerivationStep{
			{Index: 1, Title: "建立速度关系", Content: "使用 v = v0 + at。"},
			{Index: 2, Title: "建立位移关系", Content: "使用 x = x0 + v0t + 1/2 at²。"},
		}
	case domain.ModelUniformMotion:
		return []domain.DerivationStep{
			{Index: 1, Title: "建立运动关系", Content: "使用匀速直线运动公式 x = x0 + vt。"},
		}
	case domain.ModelWorkEnergy:
		return []domain.DerivationStep{
			{Index: 1, Title: "识别做功过程", Content: "明确外力做功与系统机械能变化的对应关系。"},
			{Index: 2, Title: "应用动能定理", Content: "使用 W = ΔEk 建立数量关系。"},
		}
	case domain.ModelSpringOscillator:
		return []domain.DerivationStep{
			{Index: 1, Title: "建立回复力关系", Content: "使用胡克定律 F = -kx 表示回复力。"},
			{Index: 2, Title: "求解周期或能量", Content: "结合 T = 2π√(m/k) 或弹性势能公式。"},
		}
	case domain.ModelTwoBodyMotion:
		return []domain.DerivationStep{
			{Index: 1, Title: "识别相互作用", Content: "确定两物体之间的距离、质量与引力关系。"},
			{Index: 2, Title: "建立轨道或受力方程", Content: "使用万有引力公式与圆周运动条件联立分析。"},
		}
	}

	return nil
}

func parameterSchema(modelType domain.ModelType) []domain.ParameterSchema {
	switch modelType {
	case domain.ModelProjectileMotion:
		return []domain.ParameterSchema{
			{Name: "v0", Label: "初速度", Default: 20, Min: 1, Max: 100, Step: 1, Unit: "m/s"},
			{Name: "angle_deg", Label: "抛射角", Default: 45, Min: 1, Max: 89, Step: 1, Unit: "°"},
			{Name: "t", Label: "时间", Default: 2, Min: 0.1, Max: 10, Step: 0.1, Unit: "s"},
		}
	case domain.ModelNewtonSecondLaw:
		return []domain.ParameterSchema{
			{Name: "m", Label: "质量", Default: 2, Min: 0.1, Max: 100, Step: 0.1, Unit: "kg"},
			{Name: "a", Label: "加速度", Default: 3, Min: 0.1, Max: 50, Step: 0.1, Unit: "m/s²"},
		}
	case domain.ModelUniformAcceleration:
		return []domain.ParameterSchema{
			{Name: "x0", Label: "初始位移", Default: 0, Min: -100, Max: 100, Step: 1, Unit: "m"},
			{Name: "v0", Label: "初速度", Default: 0, Min: -50, Max: 50, Step: 1, Unit: "m/s"},
			{Name: "a", Label: "加速度", Default: 2, Min: -20, Max: 20, Step: 0.5, Unit: "m/s²"},
			{Name: "t", Label: "时间", Default: 5, Min: 0.1, Max: 20, Step: 0.1, Unit: "s"},
		}
	case domain.ModelUniformMotion:
		return []domain.ParameterSchema{
			{Name: "x0", Label: "初始位移", Default: 0, Min: -100, Max: 100, Step: 1, Unit: "m"},
			{Name: "v", Label: "速度", Default: 5, Min: -50, Max: 50, Step: 1, Unit: "m/s"},
			{Name: "t", Label: "时间", Default: 5, Min: 0.1, Max: 20, Step: 0.1, Unit: "s"},
		}
	case domain.ModelWorkEnergy:
		return []domain.ParameterSchema{
			{Name: "m", Label: "质量", Default: 1, Min: 0.1, Max: 100, Step: 0.1, Unit: "kg"},
			{Name: "v", Label: "速度", Default: 2, Min: 0.1, Max: 100, Step: 0.1, Unit: "m/s"},
		}
	case domain.ModelSpringOscillator:
		return []domain.ParameterSchema{
			{Name: "k", Label: "劲度系数", Default: 20, Min: 0.1, Max: 500, Step: 0.1, Unit: "N/m"},
			{Name: "m", Label: "质量", Default: 1, Min: 0.1, Max: 100, Step: 0.1, Unit: "kg"},
			{Name: "x", Label: "位移", Default: 0.2, Min: -2, Max: 2, Step: 0.01, Unit: "m"},
		}
	case domain.ModelTwoBodyMotion:
		return []domain.ParameterSchema{
			{Name: "m1", Label: "物体一质量", Default: 5.97e24, Min: 1e10, Max: 1e30, Step: 1e10, Unit: "kg"},
			{Name: "m2", Label: "物体二质量", Default: 7.35e22, Min: 1e10, Max: 1e30, Step: 1e10, Unit: "kg"},
			{Name: "r", Label: "中心距离", Default: 3.84e8, Min: 1e3, Max: 1e12, Step: 1e3, Unit: "m"},
		}
	}

	return nil
}

func defaultParameters(modelType domain.ModelType) map[string]float64 {
	params := map[string]float64{}
	for _, item := range parameterSchema(modelType) {
		params[item.Name] = item.Default
	}
	return params
}

func resultSummary(modelType domain.ModelType, values map[string]float64) string {
	if len(values) == 0 {
		return fmt.Sprintf("已识别为 %s，但缺少足够参数完成数值计算。", modelType)
	}
	parts := make([]string, 0, len(values))
	for key, value := range values {
		parts = append(parts, fmt.Sprintf("%s=%.2f", key, value))
	}
	return fmt.Sprintf("识别为 %s，计算结果：%s。", modelType, strings.Join(parts, "，"))
}

func containsAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	return false
}
