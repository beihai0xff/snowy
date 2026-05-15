package service

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/beihai0xff/snowy/internal/modeling/physics/calculator"
	"github.com/beihai0xff/snowy/internal/modeling/physics/domain"
	physicsvalidator "github.com/beihai0xff/snowy/internal/modeling/physics/validator"
	"github.com/beihai0xff/snowy/internal/repo/llm"
)

type Option func(*serviceImpl)

type serviceImpl struct {
	calculator    calculator.Calculator
	llmChain      llm.Provider
	codeValidator physicsvalidator.CodeValidator
}

const (
	scenePhysicsOrbit3D      = "physics_orbit_3d"
	scenePhysicsSpring3D     = "physics_spring_3d"
	scenePhysicsCollision3D  = "physics_collision_3d"
	scenePhysicsProjectile3D = "physics_projectile_3d"
	scenePhysicsProjectile2D = "physics_projectile_2d"
	scenePhysicsForce3D      = "physics_force_3d"
	scenePhysicsForceDiagram = "physics_force_diagram"
	scenePhysicsGeneric3D    = "physics_generic_3d"
	scenePhysicsMotion2D     = "physics_motion_2d"

	labelAcceleration     = "加速度"
	labelInitialPosition  = "初始位移"
	unitMetersPerSecond   = "m/s"
	unitMetersPerSecond2  = "m/s²"
	unitDemoDimensionless = "演示单位"

	propCameraPitch   = "camera_pitch"
	propCameraYaw     = "camera_yaw"
	propTrailLength   = "trail_length"
	propViewDimension = "view_dimension"
)

// WithLLMProvider injects the ordered OpenAI-compatible model chain.
func WithLLMProvider(provider llm.Provider) Option {
	return func(s *serviceImpl) {
		s.llmChain = provider
	}
}

// WithCodeValidator 注入代码校验器。
func WithCodeValidator(v physicsvalidator.CodeValidator) Option {
	return func(s *serviceImpl) {
		s.codeValidator = v
	}
}

// NewService 创建物理建模服务。
func NewService(calc calculator.Calculator, opts ...Option) PhysicsService {
	if calc == nil {
		calc = calculator.NewSimpleCalculator()
	}

	svc := &serviceImpl{
		calculator:    calc,
		codeValidator: physicsvalidator.NewDefaultCodeValidator(),
	}
	for _, opt := range opts {
		opt(svc)
	}

	return svc
}

func (s *serviceImpl) Analyze(_ context.Context, question string, sessionContext string) (*domain.PhysicsModel, error) {
	question = sanitizeQuestionText(question)
	if question == "" {
		return nil, errors.New("question is empty")
	}

	fullText := strings.TrimSpace(question + " " + sessionContext)
	modelType := inferModelType(fullText)
	sceneType := inferSceneType(fullText, modelType)

	conditions, params := extractConditions(question, modelType)

	defaultProps := mergeDefaultProps(modelType, params, sceneType)

	computeResult, err := s.calculator.Compute(modelType, defaultProps)
	if err != nil {
		computeResult = &domain.ComputeResult{Warnings: []string{err.Error()}}
	}

	warnings := append([]string(nil), computeResult.Warnings...)
	if len(conditions) == 0 {
		warnings = append(warnings, "题干未抽取到完整数值，已根据默认参数模板补全预览参数")
	}

	if strings.Contains(sceneType, "3d") {
		warnings = append(warnings, "当前 3D 预览由本地 Rapier 3D 原生物理引擎驱动，大模型仅用于解析与讲解")
	}

	return &domain.PhysicsModel{
		ModelType:     modelType,
		Conditions:    conditions,
		Steps:         derivationSteps(modelType),
		ResultSummary: resultSummary(modelType, computeResult.Values),
		Warnings:      warnings,
		Parameters:    parameterSchema(modelType),
		SceneSpec: &domain.SceneSpec{
			SceneType:    sceneType,
			Title:        sceneTitle(sceneType, modelType),
			Summary:      sceneSummary(sceneType, modelType, computeResult.Values),
			RenderMode:   domain.RenderModeHTMLIframe,
			DefaultProps: defaultProps,
		},
	}, nil
}

func (s *serviceImpl) Simulate(
	_ context.Context,
	modelType domain.ModelType,
	params map[string]float64,
) (*domain.ComputeResult, error) {
	return s.calculator.Compute(modelType, params)
}

func inferModelType(text string) domain.ModelType {
	lower := strings.ToLower(text)
	for _, rule := range []struct {
		model    domain.ModelType
		keywords []string
	}{
		{model: domain.ModelCollisionMotion, keywords: []string{"碰撞", "动量", "反弹", "collision", "momentum", "elastic collision"}},
		{model: domain.ModelProjectileMotion, keywords: []string{"平抛", "抛", "projectile"}},
		{model: domain.ModelNewtonSecondLaw, keywords: []string{"牛顿", "force", "受力"}},
		{model: domain.ModelSpringOscillator, keywords: []string{"弹簧", "振子", "oscillat", "简谐", "spring"}},
		{model: domain.ModelTwoBodyMotion, keywords: []string{"双体", "天体", "行星", "卫星", "万有引力", "引力", "orbit", "轨道", "gravity"}},
		{model: domain.ModelUniformAcceleration, keywords: []string{labelAcceleration, "acceler", "匀变速"}},
		{model: domain.ModelWorkEnergy, keywords: []string{"功", "energy", "能量"}},
	} {
		if containsAny(lower, rule.keywords...) {
			return rule.model
		}
	}

	return domain.ModelUniformMotion
}

func inferSceneType(text string, modelType domain.ModelType) string {
	lower := strings.ToLower(text)
	wants3D := containsAny(lower, "3d", "三维", "立体", "空间", "spatial", "轨迹观察")

	switch {
	case modelType == domain.ModelTwoBodyMotion:
		return scenePhysicsOrbit3D
	case modelType == domain.ModelSpringOscillator:
		return scenePhysicsSpring3D
	case modelType == domain.ModelCollisionMotion:
		return scenePhysicsCollision3D
	case wants3D && modelType == domain.ModelProjectileMotion:
		return scenePhysicsProjectile3D
	case modelType == domain.ModelNewtonSecondLaw:
		return scenePhysicsForce3D
	case wants3D:
		return scenePhysicsGeneric3D
	case modelType == domain.ModelProjectileMotion:
		return scenePhysicsProjectile2D
	default:
		return scenePhysicsMotion2D
	}
}

//nolint:cyclop,gocognit,gocyclo,funlen,maintidx // Extraction rules are intentionally grouped by model.
func extractConditions(question string, modelType domain.ModelType) ([]domain.Condition, map[string]float64) {
	text := normalizeQuestion(question)
	params := map[string]float64{}
	conditions := make([]domain.Condition, 0, 6)

	add := func(name string, value float64, unit string) {
		if _, ok := params[name]; ok {
			return
		}

		params[name] = value
		conditions = append(conditions, domain.Condition{Name: name, Value: value, Unit: unit})
	}

	if v, ok := captureNamedFloat(text, `(?:质量|mass)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("m", v, "kg")
	}

	if _, ok := params["m"]; !ok {
		if v, ok := captureFloat(text, `([0-9]+(?:\.[0-9]+)?)\s*kg`); ok {
			add("m", v, "kg")
		}
	}

	velocityName := "v0"
	if modelType == domain.ModelUniformMotion || modelType == domain.ModelWorkEnergy {
		velocityName = "v"
	}

	if modelType == domain.ModelCollisionMotion || modelType == domain.ModelTwoBodyMotion ||
		modelType == domain.ModelSpringOscillator {
		velocityName = ""
	}

	if velocityName != "" {
		if v, ok := captureNamedFloat(text, `(?:初速度|速度|v0|速率)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
			add(velocityName, v, "m/s")
		}

		if _, ok := params[velocityName]; !ok {
			if v, ok := captureFloat(text, `([0-9]+(?:\.[0-9]+)?)\s*m/s`); ok {
				add(velocityName, v, "m/s")
			}
		}
	}

	if v, ok := captureNamedFloat(text, `(?:抛射角|角度|夹角|angle)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("angle_deg", v, "°")
	}

	if _, ok := params["angle_deg"]; !ok {
		if v, ok := captureFloat(text, `([0-9]+(?:\.[0-9]+)?)\s*(?:°|度)`); ok {
			add("angle_deg", v, "°")
		}
	}

	if v, ok := captureNamedFloat(text, `(?:时间|经过|历时|t)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("t", v, "s")
	}

	if _, ok := params["t"]; !ok {
		if v, ok := captureFloat(text, `([0-9]+(?:\.[0-9]+)?)\s*秒`); ok {
			add("t", v, "s")
		}
	}

	if v, ok := captureNamedFloat(text, `(?:加速度|acceleration|a)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("a", v, "m/s²")
	}

	if _, ok := params["a"]; !ok {
		if v, ok := captureFloat(text, `([0-9]+(?:\.[0-9]+)?)\s*m/s(?:²|\^2)`); ok {
			add("a", v, "m/s²")
		}
	}

	if v, ok := captureNamedFloat(text, `(?:初始位移|位移起点|x0)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("x0", v, "m")
	}

	if v, ok := captureNamedFloat(text, `(?:高度|抛出高度|竖直高度|height|h)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("h", v, "m")
	}

	if _, ok := params["h"]; !ok && modelType == domain.ModelProjectileMotion {
		if v, ok := captureFloat(text, `([0-9]+(?:\.[0-9]+)?)\s*(?:m|米)(?:高|高度)?`); ok {
			add("h", v, "m")
		}
	}

	if v, ok := captureNamedFloat(text, `(?:目标距离|目标水平距离|target_x|靶距)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("target_x", v, "m")
	}

	if v, ok := captureNamedFloat(text, `(?:重力加速度|g)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("g", v, "m/s²")
	}

	if v, ok := captureNamedFloat(text, `(?:劲度系数|k)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("k", v, "N/m")
	}

	if v, ok := captureNamedFloat(text, `(?:位移|伸长量|x)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("x", v, "m")
	}

	if v, ok := captureNamedFloat(text, `(?:物体一质量|质量1|m1)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("m1", v, "kg")
	}

	if v, ok := captureNamedFloat(text, `(?:中心质量|central_mass)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("central_mass", v, "演示单位")
	}

	if v, ok := captureNamedFloat(text, `(?:物体二质量|质量2|m2)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("m2", v, "kg")
	}

	if v, ok := captureNamedFloat(text, `(?:卫星质量|satellite_mass)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("satellite_mass", v, "演示单位")
	}

	if v, ok := captureNamedFloat(text, `(?:半径|距离|轨道半径|orbit_radius|r)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		if modelType == domain.ModelTwoBodyMotion {
			add("orbit_radius", v, "演示单位")
		} else {
			add("r", v, "m")
		}
	}

	if v, ok := captureNamedFloat(text, `(?:切向速度|tangential_speed)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("tangential_speed", v, "演示单位/s")
	}

	if v, ok := captureNamedFloat(text, `(?:引力强度|gravitational_strength)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("gravitational_strength", v, "")
	}

	if v, ok := captureNamedFloat(text, `(?:偏心率|eccentricity)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("eccentricity", v, "")
	}

	if v, ok := captureNamedFloat(text, `(?:速度1|v1)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("v1", v, "m/s")
	}

	if v, ok := captureNamedFloat(text, `(?:速度2|v2)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("v2", v, "m/s")
	}

	if v, ok := captureNamedFloat(text, `(?:恢复系数|反弹系数|restitution)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("restitution", v, "")
	}

	// 如果仍然没有明显参数，按数值顺序回退，但避免把单位明显不匹配的值映射错位。
	if len(params) == 0 {
		matches := regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s*([a-zA-Z°/%μ²\^/]*)`).FindAllStringSubmatch(text, -1)
		fallbackKeys := []string{"v0", "angle_deg", "t", "a", "m"}

		for i, match := range matches {
			if len(match) < 2 {
				continue
			}

			value, err := strconv.ParseFloat(match[1], 64)
			if err != nil {
				continue
			}

			name := fmt.Sprintf("value_%d", i+1)
			if i < len(fallbackKeys) {
				name = fallbackKeys[i]
			}

			add(name, value, strings.TrimSpace(match[2]))
		}
	}

	return conditions, params
}

func derivationSteps(modelType domain.ModelType) []domain.DerivationStep {
	switch modelType {
	case domain.ModelProjectileMotion:
		return []domain.DerivationStep{
			{Index: 1, Title: "分解初速度", Content: "将初速度分解为水平和竖直两个方向，确定抛射角对应的分量。"},
			{Index: 2, Title: "建立位移关系", Content: "水平做匀速运动，竖直做匀变速运动，分别建立位移方程。"},
			{Index: 3, Title: "运行原生物理仿真", Content: "将速度、时间、角度等量注入 Rapier 3D 原生物理引擎，在浏览器中播放轨迹、速度箭头和关键点。"},
		}
	case domain.ModelNewtonSecondLaw:
		return []domain.DerivationStep{
			{Index: 1, Title: "识别受力", Content: "明确研究对象、受力方向及合外力。"},
			{Index: 2, Title: "应用牛顿第二定律", Content: "建立 F = ma 的数量关系，并提取可视化参数。"},
			{Index: 3, Title: "运行 3D 受力仿真", Content: "把质量、加速度和合力映射到 Rapier 3D 刚体、地面、坐标轴和可切换 2D/3D 的矢量预览。"},
		}
	case domain.ModelUniformAcceleration:
		return []domain.DerivationStep{
			{Index: 1, Title: "建立速度关系", Content: "使用 v = v0 + at 确定速度随时间变化。"},
			{Index: 2, Title: "建立位移关系", Content: "使用 x = x0 + v0t + 1/2 at² 计算位移。"},
			{Index: 3, Title: "运行运动预览", Content: "把位移、速度和时间映射到本地物理引擎预览中的动画轨迹。"},
		}
	case domain.ModelUniformMotion:
		return []domain.DerivationStep{
			{Index: 1, Title: "建立运动关系", Content: "使用匀速直线运动公式 x = x0 + vt，并在本地物理引擎预览中展示位移示意。"},
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
			{Index: 1, Title: "识别相互作用", Content: "确定中心天体、环绕物体、距离和轨道速度等关键量。"},
			{Index: 2, Title: "建立轨道关系", Content: "用万有引力提供向心作用的思路分析轨道，并映射为归一化教学演示参数。"},
			{Index: 3, Title: "运行轨道演示", Content: "在本地物理引擎预览中展示发光天体、轨道尾迹、速度和引力方向。"},
		}
	case domain.ModelCollisionMotion:
		return []domain.DerivationStep{
			{Index: 1, Title: "识别碰撞对象", Content: "确定两个物体的质量、初速度和碰撞方向。"},
			{Index: 2, Title: "应用动量关系", Content: "用动量守恒和恢复系数估计碰撞前后速度变化。"},
			{Index: 3, Title: "运行碰撞演示", Content: "在本地物理引擎中播放碰撞、反弹、速度箭头和能量变化面板。"},
		}
	}

	return nil
}

func parameterSchema(modelType domain.ModelType) []domain.ParameterSchema {
	switch modelType {
	case domain.ModelProjectileMotion:
		return []domain.ParameterSchema{
			{Name: "v0", Label: "初速度", Default: 20, Min: 1, Max: 100, Step: 1, Unit: unitMetersPerSecond},
			{Name: "h", Label: "抛出高度", Default: 20, Min: 1, Max: 100, Step: 1, Unit: "m"},
			{Name: "g", Label: "重力加速度", Default: 9.8, Min: 1, Max: 20, Step: 0.1, Unit: unitMetersPerSecond2},
			{Name: "target_x", Label: "目标水平距离", Default: 40, Min: 5, Max: 180, Step: 1, Unit: "m"},
			{Name: "angle_deg", Label: "抛射角", Default: 0, Min: 0, Max: 80, Step: 1, Unit: "°"},
		}
	case domain.ModelNewtonSecondLaw:
		return []domain.ParameterSchema{
			{Name: "m", Label: "质量", Default: 2, Min: 0.1, Max: 100, Step: 0.1, Unit: "kg"},
			{Name: "a", Label: labelAcceleration, Default: 3, Min: 0.1, Max: 50, Step: 0.1, Unit: unitMetersPerSecond2},
		}
	case domain.ModelUniformAcceleration:
		return []domain.ParameterSchema{
			{Name: "x0", Label: labelInitialPosition, Default: 0, Min: -100, Max: 100, Step: 1, Unit: "m"},
			{Name: "v0", Label: "初速度", Default: 0, Min: -50, Max: 50, Step: 1, Unit: unitMetersPerSecond},
			{Name: "a", Label: labelAcceleration, Default: 2, Min: -20, Max: 20, Step: 0.5, Unit: unitMetersPerSecond2},
			{Name: "t", Label: "时间", Default: 5, Min: 0.1, Max: 20, Step: 0.1, Unit: "s"},
		}
	case domain.ModelUniformMotion:
		return []domain.ParameterSchema{
			{Name: "x0", Label: labelInitialPosition, Default: 0, Min: -100, Max: 100, Step: 1, Unit: "m"},
			{Name: "v", Label: "速度", Default: 5, Min: -50, Max: 50, Step: 1, Unit: unitMetersPerSecond},
			{Name: "t", Label: "时间", Default: 5, Min: 0.1, Max: 20, Step: 0.1, Unit: "s"},
		}
	case domain.ModelWorkEnergy:
		return []domain.ParameterSchema{
			{Name: "m", Label: "质量", Default: 1, Min: 0.1, Max: 100, Step: 0.1, Unit: "kg"},
			{Name: "v", Label: "速度", Default: 2, Min: 0.1, Max: 100, Step: 0.1, Unit: unitMetersPerSecond},
		}
	case domain.ModelSpringOscillator:
		return []domain.ParameterSchema{
			{Name: "k", Label: "劲度系数", Default: 24, Min: 1, Max: 80, Step: 1, Unit: "N/m"},
			{Name: "m", Label: "质量", Default: 1.2, Min: 0.2, Max: 8, Step: 0.1, Unit: "kg"},
			{Name: "x", Label: labelInitialPosition, Default: 1.4, Min: -3, Max: 3, Step: 0.05, Unit: "m"},
			{Name: "damping", Label: "阻尼", Default: 0.18, Min: 0, Max: 2, Step: 0.02, Unit: ""},
		}
	case domain.ModelTwoBodyMotion:
		return []domain.ParameterSchema{
			{Name: "central_mass", Label: "中心质量", Default: 8, Min: 1, Max: 20, Step: 0.1, Unit: unitDemoDimensionless},
			{
				Name:    "satellite_mass",
				Label:   "卫星质量",
				Default: 1,
				Min:     0.1,
				Max:     6,
				Step:    0.1,
				Unit:    unitDemoDimensionless,
			},
			{
				Name:    "orbit_radius",
				Label:   "轨道半径",
				Default: 3.6,
				Min:     1.4,
				Max:     6.5,
				Step:    0.1,
				Unit:    unitDemoDimensionless,
			},
			{Name: "tangential_speed", Label: "切向速度", Default: 2.25, Min: 0.3, Max: 5.5, Step: 0.05, Unit: "演示单位/s"},
			{Name: "eccentricity", Label: "偏心率", Default: 0.18, Min: 0, Max: 0.75, Step: 0.01, Unit: ""},
			{Name: "gravitational_strength", Label: "引力强度", Default: 10, Min: 1, Max: 24, Step: 0.1, Unit: ""},
		}
	case domain.ModelCollisionMotion:
		return []domain.ParameterSchema{
			{Name: "m1", Label: "物体一质量", Default: 1.5, Min: 0.2, Max: 8, Step: 0.1, Unit: "kg"},
			{Name: "m2", Label: "物体二质量", Default: 1, Min: 0.2, Max: 8, Step: 0.1, Unit: "kg"},
			{Name: "v1", Label: "物体一速度", Default: 4.5, Min: -8, Max: 8, Step: 0.1, Unit: unitMetersPerSecond},
			{Name: "v2", Label: "物体二速度", Default: -2.5, Min: -8, Max: 8, Step: 0.1, Unit: unitMetersPerSecond},
			{Name: "restitution", Label: "恢复系数", Default: 0.9, Min: 0, Max: 1, Step: 0.01, Unit: ""},
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
		parts = append(parts, fmt.Sprintf("%s=%s", key, formatResultValue(value)))
	}

	return fmt.Sprintf("识别为 %s，计算结果：%s。", modelType, strings.Join(parts, "，"))
}

func formatResultValue(value float64) string {
	abs := math.Abs(value)
	if abs > 0 && (abs >= 100000 || abs < 0.01) {
		return fmt.Sprintf("%.3g", value)
	}

	return fmt.Sprintf("%.2f", value)
}

//nolint:cyclop,gocognit // Defaults are a compact model-specific table with minor derived values.
func mergeDefaultProps(modelType domain.ModelType, params map[string]float64, sceneType string) map[string]float64 {
	props := defaultParameters(modelType)
	maps.Copy(props, params)

	if modelType == domain.ModelProjectileMotion {
		if _, ok := props["g"]; !ok {
			props["g"] = 9.8
		}
	}

	if sceneType == scenePhysicsGeneric3D {
		if _, ok := props["size"]; !ok {
			props["size"] = 110
		}

		if _, ok := props["rotation_speed"]; !ok {
			props["rotation_speed"] = 0.018
		}
	}

	if sceneType == scenePhysicsOrbit3D {
		for key, value := range map[string]float64{propViewDimension: 3, propTrailLength: 240, propCameraYaw: 0.72, propCameraPitch: 0.54} {
			if _, ok := props[key]; !ok {
				props[key] = value
			}
		}
	}

	if sceneType == scenePhysicsSpring3D {
		for key, value := range map[string]float64{propViewDimension: 3, propTrailLength: 180, propCameraYaw: 0.6, propCameraPitch: 0.38} {
			if _, ok := props[key]; !ok {
				props[key] = value
			}
		}
	}

	if sceneType == scenePhysicsCollision3D {
		for key, value := range map[string]float64{propViewDimension: 3, propTrailLength: 200, propCameraYaw: 0.45, propCameraPitch: 0.38} {
			if _, ok := props[key]; !ok {
				props[key] = value
			}
		}
	}

	if sceneType == scenePhysicsForce3D {
		if _, ok := props[propViewDimension]; !ok {
			props[propViewDimension] = 3
		}

		if _, ok := props[propCameraYaw]; !ok {
			props[propCameraYaw] = 0.55
		}

		if _, ok := props[propCameraPitch]; !ok {
			props[propCameraPitch] = 0.42
		}
	}

	return props
}

//nolint:cyclop // Titles mirror the supported scene/model matrix.
func sceneTitle(sceneType string, modelType domain.ModelType) string {
	switch sceneType {
	case scenePhysicsOrbit3D:
		return "天体轨道 3D 动态演示"
	case scenePhysicsSpring3D:
		return "弹簧振子 3D 动态演示"
	case scenePhysicsCollision3D:
		return "碰撞运动 3D 动态演示"
	case scenePhysicsProjectile3D:
		return "平抛运动 3D 轨迹预览"
	case scenePhysicsProjectile2D:
		return "平抛运动浏览器轨迹预览"
	case scenePhysicsForce3D:
		return "牛顿第二定律 3D 受力模型"
	case scenePhysicsForceDiagram:
		return "牛顿第二定律受力示意"
	case scenePhysicsGeneric3D:
		return "3D 场景浏览器渲染预览"
	case scenePhysicsMotion2D:
		return "运动场景浏览器预览"
	default:
		return fmt.Sprintf("%s 浏览器渲染预览", modelType)
	}
}

func sceneSummary(sceneType string, modelType domain.ModelType, values map[string]float64) string {
	base := resultSummary(modelType, values)

	switch sceneType {
	case scenePhysicsOrbit3D:
		return base + " 已转换为教学演示优先的 3D 天体轨道场景，支持发光星体、轨道尾迹、速度与引力箭头。"
	case scenePhysicsSpring3D:
		return base + " 已转换为 3D 弹簧振子场景，支持发光弹簧、回复力箭头、能量条和阻尼调节。"
	case scenePhysicsCollision3D:
		return base + " 已转换为 3D 碰撞演示场景，支持双刚体反弹、速度箭头、轨迹残影和碰撞闪光。"
	case scenePhysicsProjectile3D:
		return base + " 已转换为 Rapier 3D 原生物理引擎轨迹场景，用于观察空间投影效果。"
	case scenePhysicsProjectile2D:
		return base + " 已转换为本地物理引擎中的可交互轨迹预览。"
	case scenePhysicsForce3D:
		return base + " 已转换为 Rapier 3D 原生受力模型，支持方块刚体、地面、坐标轴、力矢量、加速度矢量和 2D/3D 切换。"
	case scenePhysicsForceDiagram:
		return base + " 已转换为本地物理引擎中的受力箭头与加速度示意。"
	case scenePhysicsGeneric3D:
		return base + " 已转换为 Rapier 3D 原生物理引擎通用场景。"
	default:
		return base + " 已转换为本地物理引擎中的交互演示。"
	}
}

func normalizeQuestion(question string) string {
	replacer := strings.NewReplacer("／", "/", "㎡", "m", "﹣", "-", "，", ",", "：", ":")

	return sanitizeQuestionText(replacer.Replace(question))
}

func sanitizeQuestionText(question string) string {
	text := strings.TrimSpace(question)
	if text == "" {
		return text
	}

	// View-mode hints such as “3D/2D” are useful for scene selection but should
	// never become physics conditions like v0=3.  Strip them before numeric
	// extraction while keeping semantic words such as “三维/空间”.
	re := regexp.MustCompile(`(?i)(^|[^a-z0-9])([23])\s*d([^a-z0-9]|$)`)

	return strings.TrimSpace(re.ReplaceAllString(text, "${1}${3}"))
}

func captureNamedFloat(text string, pattern string) (float64, bool) {
	return captureFloat(text, pattern)
}

func captureFloat(text string, pattern string) (float64, bool) {
	re := regexp.MustCompile(pattern)

	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return 0, false
	}

	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, false
	}

	return value, true
}

func containsAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	return false
}

func cloneNumberMap(src map[string]float64) map[string]float64 {
	if len(src) == 0 {
		return map[string]float64{}
	}

	dst := make(map[string]float64, len(src))
	maps.Copy(dst, src)

	return dst
}
