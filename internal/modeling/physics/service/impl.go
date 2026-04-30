package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/beihai0xff/snowy/internal/modeling/physics/calculator"
	"github.com/beihai0xff/snowy/internal/modeling/physics/domain"
	physicsvalidator "github.com/beihai0xff/snowy/internal/modeling/physics/validator"
	"github.com/beihai0xff/snowy/internal/repo/llm"
)

var numberPattern = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)`)

type Option func(*serviceImpl)

type serviceImpl struct {
	calculator    calculator.Calculator
	primaryLLM    llm.Provider
	fallbackLLM   llm.Provider
	codeValidator physicsvalidator.CodeValidator
}

// WithLLMProviders 注入主/备选模型提供方。
func WithLLMProviders(primary, fallback llm.Provider) Option {
	return func(s *serviceImpl) {
		s.primaryLLM = primary
		s.fallbackLLM = fallback
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
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, errors.New("question is empty")
	}

	fullText := strings.TrimSpace(question + " " + sessionContext)
	modelType := inferModelType(fullText)
	sceneType := inferSceneType(fullText, modelType)

	conditions, params := extractConditions(question, modelType)
	defaultProps := mergeDefaultProps(modelType, params, sceneType)
	if len(conditions) == 0 {
		params = cloneNumberMap(defaultProps)
	}

	computeResult, err := s.calculator.Compute(modelType, defaultProps)
	if err != nil {
		computeResult = &domain.ComputeResult{Warnings: []string{err.Error()}}
	}

	warnings := append([]string(nil), computeResult.Warnings...)
	if len(conditions) == 0 {
		warnings = append(warnings, "题干未抽取到完整数值，已根据默认参数模板补全预览参数")
	}
	if strings.Contains(sceneType, "3d") {
		warnings = append(warnings, "当前 3D 效果为浏览器轻量渲染示意，不依赖原生物理引擎")
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

func inferSceneType(text string, modelType domain.ModelType) string {
	lower := strings.ToLower(text)
	wants3D := containsAny(lower, "3d", "三维", "立体", "空间", "spatial", "轨迹观察")

	switch {
	case wants3D && modelType == domain.ModelProjectileMotion:
		return "physics_projectile_3d"
	case modelType == domain.ModelNewtonSecondLaw:
		return "physics_force_3d"
	case wants3D:
		return "physics_generic_3d"
	case modelType == domain.ModelProjectileMotion:
		return "physics_projectile_2d"
	default:
		return "physics_motion_2d"
	}
}

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
	if modelType == domain.ModelUniformMotion {
		velocityName = "v"
	}

	if v, ok := captureNamedFloat(text, `(?:初速度|速度|v0|速率)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add(velocityName, v, "m/s")
	}
	if _, ok := params[velocityName]; !ok {
		if v, ok := captureFloat(text, `([0-9]+(?:\.[0-9]+)?)\s*m/s`); ok {
			add(velocityName, v, "m/s")
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
	if v, ok := captureNamedFloat(text, `(?:劲度系数|k)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("k", v, "N/m")
	}
	if v, ok := captureNamedFloat(text, `(?:位移|伸长量|x)\s*[:：=]?\s*([0-9]+(?:\.[0-9]+)?)`); ok {
		add("x", v, "m")
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
			{Index: 3, Title: "组织渲染场景", Content: "将速度、时间、角度等量注入前端代码，在浏览器中绘制轨迹与关键点。"},
		}
	case domain.ModelNewtonSecondLaw:
		return []domain.DerivationStep{
			{Index: 1, Title: "识别受力", Content: "明确研究对象、受力方向及合外力。"},
			{Index: 2, Title: "应用牛顿第二定律", Content: "建立 F = ma 的数量关系，并提取可视化参数。"},
			{Index: 3, Title: "生成 3D 受力模型代码", Content: "把质量、加速度和合力映射到原生 WebGL 方块、地面网格、坐标轴和可切换 2D/3D 的矢量预览。"},
		}
	case domain.ModelUniformAcceleration:
		return []domain.DerivationStep{
			{Index: 1, Title: "建立速度关系", Content: "使用 v = v0 + at 确定速度随时间变化。"},
			{Index: 2, Title: "建立位移关系", Content: "使用 x = x0 + v0t + 1/2 at² 计算位移。"},
			{Index: 3, Title: "生成运动预览", Content: "把位移、速度和时间映射到浏览器中的动画轨迹。"},
		}
	case domain.ModelUniformMotion:
		return []domain.DerivationStep{{Index: 1, Title: "建立运动关系", Content: "使用匀速直线运动公式 x = x0 + vt，并生成浏览器中的位移示意。"}}
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

func mergeDefaultProps(modelType domain.ModelType, params map[string]float64, sceneType string) map[string]float64 {
	props := defaultParameters(modelType)
	for key, value := range params {
		props[key] = value
	}

	if modelType == domain.ModelProjectileMotion {
		if _, ok := props["g"]; !ok {
			props["g"] = 9.8
		}
	}

	if sceneType == "physics_generic_3d" {
		if _, ok := props["size"]; !ok {
			props["size"] = 110
		}
		if _, ok := props["rotation_speed"]; !ok {
			props["rotation_speed"] = 0.018
		}
	}

	if sceneType == "physics_force_3d" {
		if _, ok := props["view_dimension"]; !ok {
			props["view_dimension"] = 3
		}
		if _, ok := props["camera_yaw"]; !ok {
			props["camera_yaw"] = 0.55
		}
		if _, ok := props["camera_pitch"]; !ok {
			props["camera_pitch"] = 0.42
		}
	}

	return props
}

func sceneTitle(sceneType string, modelType domain.ModelType) string {
	switch sceneType {
	case "physics_projectile_3d":
		return "平抛运动 3D 轨迹预览"
	case "physics_projectile_2d":
		return "平抛运动浏览器轨迹预览"
	case "physics_force_3d":
		return "牛顿第二定律 3D 受力模型"
	case "physics_force_diagram":
		return "牛顿第二定律受力示意"
	case "physics_generic_3d":
		return "3D 场景浏览器渲染预览"
	case "physics_motion_2d":
		return "运动场景浏览器预览"
	default:
		return fmt.Sprintf("%s 浏览器渲染预览", modelType)
	}
}

func sceneSummary(sceneType string, modelType domain.ModelType, values map[string]float64) string {
	base := resultSummary(modelType, values)
	switch sceneType {
	case "physics_projectile_3d":
		return base + " 已转换为 3D 轨迹浏览器场景，用于观察空间投影效果。"
	case "physics_projectile_2d":
		return base + " 已转换为浏览器中的可交互轨迹代码。"
	case "physics_force_3d":
		return base + " 已转换为原生 WebGL 3D 受力模型代码，支持方块、地面网格、坐标轴、力矢量、加速度矢量和 2D/3D 切换。"
	case "physics_force_diagram":
		return base + " 已转换为浏览器中的受力箭头与加速度示意代码。"
	case "physics_generic_3d":
		return base + " 已转换为轻量 3D 浏览器场景代码。"
	default:
		return base + " 已转换为浏览器中的交互演示代码。"
	}
}

func normalizeQuestion(question string) string {
	replacer := strings.NewReplacer("／", "/", "㎡", "m", "﹣", "-", "，", ",", "：", ":")
	return replacer.Replace(question)
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
	for key, value := range src {
		dst[key] = value
	}

	return dst
}
