package generative

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	biologydomain "github.com/beihai0xff/snowy/internal/modeling/biology/domain"
	physicsdomain "github.com/beihai0xff/snowy/internal/modeling/physics/domain"
	"github.com/beihai0xff/snowy/internal/repo/llm"
)

type fakeLLM struct {
	name       string
	model      string
	generateFn func(context.Context, *llm.Request) (*llm.Response, error)
}

func (f fakeLLM) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	return f.generateFn(ctx, req)
}
func (f fakeLLM) GenerateStream(context.Context, *llm.Request, chan<- llm.StreamChunk) error {
	return nil
}
func (f fakeLLM) HealthCheck(context.Context) error { return nil }
func (f fakeLLM) EstimateCost(context.Context, *llm.Request) (*llm.Cost, error) {
	return &llm.Cost{}, nil
}
func (f fakeLLM) Name() string {
	if f.name != "" {
		return f.name
	}
	return "fake"
}
func (f fakeLLM) ConfiguredModel() string         { return f.model }
func (f fakeLLM) ConfiguredBaseURL() string       { return "" }
func (f fakeLLM) ConfiguredModelProvider() string { return "" }

type fakePhysics struct{}

func (fakePhysics) Analyze(context.Context, string, string) (*physicsdomain.PhysicsModel, error) {
	return &physicsdomain.PhysicsModel{ModelType: physicsdomain.ModelProjectileMotion, ResultSummary: "fallback physics", Parameters: []physicsdomain.ParameterSchema{{Name: "v0", Label: "初速度", Unit: "m/s", Default: 20, Min: 0, Max: 60, Step: 1}}, Steps: []physicsdomain.DerivationStep{{Index: 1, Title: "分解运动", Content: "..."}}}, nil
}
func (fakePhysics) Simulate(context.Context, physicsdomain.ModelType, map[string]float64) (*physicsdomain.ComputeResult, error) {
	return nil, nil
}
func (fakePhysics) GenerateRender(context.Context, *physicsdomain.SceneSpec, string) (*physicsdomain.RenderArtifact, error) {
	return nil, nil
}

type fakeBiology struct{}

func (fakeBiology) Analyze(context.Context, string, string) (*biologydomain.BiologyModel, error) {
	return &biologydomain.BiologyModel{Topic: "photosynthesis", Concepts: []biologydomain.Concept{{Name: "光照强度", Type: "factor"}, {Name: "有机物积累", Type: "result"}}, Relations: []biologydomain.Relation{{Source: "光照强度", Target: "有机物积累", Type: "influences"}}, ProcessSteps: []biologydomain.ProcessStep{{Index: 1, Title: "识别变量", Content: "..."}}, ExperimentVariables: &biologydomain.ExperimentVariables{Independent: []string{"光照强度"}, Dependent: []string{"有机物积累"}, Controlled: []string{"温度"}}, ResultSummary: "fallback biology"}, nil
}

func TestCompileUsesLLMJSON(t *testing.T) {
	content := `{"domain":"physics","question":"平抛","learning_model":{"domain":"physics","grade_band":"high_school","topic":"projectile","learning_goal":"理解平抛"},"evidence_refs":["平抛运动可分解为水平方向匀速直线运动和竖直方向自由落体运动"],"reasoning_trace":{"summary":"先分解运动","confidence":0.9},"generative_model":{"domain":"physics","grade_band":"high_school","topic":"projectile","learning_goal":"理解平抛","variables":[{"name":"v0","label":"初速度","unit":"m/s","default":20,"min":0,"max":60}]},"simulation_logic":{"simulation_type":"generated_projectile_2d","runtime":"safe_math_dsl","variables":[{"name":"v0","label":"初速度","unit":"m/s","default":20,"min":0,"max":60}],"formulas":[{"id":"x","expr":"x=v0*t","meaning":"水平位移"}],"render_instructions":{"coordinate_system":"2d_cartesian"},"local_recompute_allowed":true},"interaction_plan":{"regeneration_policy":{"local_recompute":["v0"],"llm_regenerate":["new_force"]}},"assessment_tasks":[],"validation_report":{"schema_valid":true},"confidence":0.91}`
	svc := NewCompilerService(nil, fakePhysics{}, fakeBiology{}, nil, WithLLMProviders(fakeLLM{name: "primary", model: "m1", generateFn: func(context.Context, *llm.Request) (*llm.Response, error) {
		return &llm.Response{Content: content, Model: "m1"}, nil
	}}, nil))
	pkg, err := svc.Compile(context.Background(), &CompileRequest{Message: "平抛运动", Domain: DomainPhysics, GradeBand: GradeBandHighSchool})
	require.NoError(t, err)
	assert.Equal(t, DomainPhysics, pkg.Domain)
	assert.Equal(t, "m1", pkg.ModelName)
	assert.False(t, pkg.ValidationReport.FallbackRequired)
	assert.Greater(t, pkg.Confidence, 0.8)
}

func TestCompileFallsBackToSecondProvider(t *testing.T) {
	content := `{"domain":"biology","question":"光合作用","learning_model":{"domain":"biology","grade_band":"high_school","topic":"photosynthesis","learning_goal":"理解光合作用"},"evidence_refs":[{"doc_id":"d1","source_type":"textbook","snippet":"光合作用","confidence":0.9}],"reasoning_trace":{"summary":"分析变量","confidence":0.9},"generative_model":{"domain":"biology","grade_band":"high_school","topic":"photosynthesis","learning_goal":"理解光合作用"},"visualization_graph":{"visualization_type":"generated_biology_process_graph","topic":"photosynthesis","nodes":[{"id":"n1","label":"光照","type":"factor"}],"edges":[],"experiment_variables":{"independent":["光照"],"dependent":["有机物"],"controlled":["温度"]}},"interaction_plan":{"regeneration_policy":{}},"assessment_tasks":[],"validation_report":{},"confidence":0.88}`
	svc := NewCompilerService(nil, fakePhysics{}, fakeBiology{}, nil, WithLLMProviders(fakeLLM{name: "primary", generateFn: func(context.Context, *llm.Request) (*llm.Response, error) { return nil, errors.New("down") }}, fakeLLM{name: "fallback", generateFn: func(context.Context, *llm.Request) (*llm.Response, error) {
		return &llm.Response{Content: content, Model: "m2"}, nil
	}}))
	pkg, err := svc.Compile(context.Background(), &CompileRequest{Message: "光合作用", Domain: DomainBiology})
	require.NoError(t, err)
	assert.Equal(t, "m2", pkg.ModelName)
	assert.Equal(t, DomainBiology, pkg.Domain)
}

func TestCompileRuleFallbackWhenLLMFails(t *testing.T) {
	svc := NewCompilerService(nil, fakePhysics{}, fakeBiology{}, nil, WithLLMProviders(fakeLLM{name: "primary", generateFn: func(context.Context, *llm.Request) (*llm.Response, error) { return nil, errors.New("down") }}, nil))
	pkg, err := svc.Compile(context.Background(), &CompileRequest{Message: "平抛运动", Domain: DomainPhysics})
	require.NoError(t, err)
	assert.Equal(t, "fallback", pkg.Status)
	assert.True(t, pkg.ValidationReport.FallbackRequired)
	assert.True(t, strings.Contains(pkg.FallbackReason, "down"))
}

func TestDecodePackageJSONNormalizesMiMoShape(t *testing.T) {
	content := `这里是结果：{"package_id":"package","domain":"physics","question":"平抛","learning_model":{"domain":"physics","grade_band":"high_school","topic":"projectile","learning_goal":"理解平抛"},"evidence_refs":["平抛运动可分解为水平方向匀速直线运动和竖直方向自由落体运动"],"reasoning_trace":{"summary":"先分解运动","confidence":0.9},"generative_model":{"domain":"physics","grade_band":"high_school","topic":"projectile","learning_goal":"理解平抛","variables":[{"name":"v0","label":"初速度","unit":"m/s","default":20,"min":0,"max":60}]},"simulation_logic":{"simulation_type":"generated_projectile_2d","runtime":"safe_math_dsl","variables":[{"name":"v0","label":"初速度","unit":"m/s","default":20,"min":0,"max":60}],"formulas":["x = v0 * t"],"render_instructions":{"coordinate_system":"2d_cartesian"},"local_recompute_allowed":true},"interaction_plan":{"regeneration_policy":{"local_recompute":["v0"],"llm_regenerate":["new_force"]}},"assessment_tasks":[],"validation_report":{"schema_valid":true},"confidence":0.91} 结束`
	content = strings.ReplaceAll(content, `\"`, `"`)
	pkg, err := decodePackageJSON(content)
	require.NoError(t, err)
	assert.NotEqual(t, "package", pkg.PackageID.String())
	require.NotNil(t, pkg.SimulationLogic)
	require.Len(t, pkg.SimulationLogic.Formulas, 1)
	assert.Equal(t, "x = v0 * t", pkg.SimulationLogic.Formulas[0].Expr)
}

func TestDecodePackageJSONNormalizesMiMoSpringShape(t *testing.T) {
	content := `{
		"package_id":"package",
		"domain":"physics",
		"question":"弹簧振子简谐运动中周期和质量、劲度系数有什么关系？",
		"learning_model":{"model_type":"inquiry_based","objectives":["理解周期与质量、劲度系数的关系"]},
		"evidence_refs":[{"doc_id":"runtime-grounding","source_type":"runtime","snippet":"高中通用知识","confidence":0.35}],
		"reasoning_trace":{"summary":"周期 T=2π√(m/k)","confidence":0.35},
		"generative_model":{"model_type":"simulation_based","description":"弹簧振子模型","parameters":["mass","stiffness"],"outputs":["period"]},
		"simulation_logic":{
			"variables":[
				{"name":"mass","symbol":"m","unit":"kg","description":"振子质量"},
				{"name":"stiffness","symbol":"k","unit":"N/m","description":"弹簧劲度系数"},
				{"name":"period","symbol":"T","unit":"s","description":"振动周期"}
			],
			"formula":"T = 2π√(m/k)",
			"rendering_instructions":"绘制周期 T 随质量 m 或劲度系数 k 变化的图表，使用滑块交互实时更新。",
			"interaction_plan":{"user_controls":["调整质量 m 的滑块（范围 0.1-10 kg）","调整劲度系数 k 的滑块（范围 10-1000 N/m）"],"real_time_update":"当 m 或 k 改变时，实时计算 T"},
			"re_reasoning_conditions":["当用户改变 m 或 k 时，重新计算 T 以验证公式关系"]
		},
		"interaction_plan":{"mode":"interactive_simulation","elements":["滑块 for m","滑块 for k"],"steps":["用户调整参数滑块","系统计算周期 T"]},
		"assessment_tasks":["写出周期公式并说明变量关系"],
		"validation_report":{"evidence_quality":"low","confidence_level":0.35},
		"regeneration_hints":["若加入阻尼，需要重新推理"],
		"warnings":["证据置信度较低"],
		"confidence":0.35
	}`
	pkg, err := decodePackageJSON(content)
	require.NoError(t, err)
	assert.Equal(t, DomainPhysics, pkg.Domain)
	assert.Equal(t, "理解周期与质量、劲度系数的关系", pkg.LearningModel.LearningGoal)
	require.NotNil(t, pkg.SimulationLogic)
	require.Len(t, pkg.SimulationLogic.Formulas, 1)
	assert.Equal(t, "T = 2π√(m/k)", pkg.SimulationLogic.Formulas[0].Expr)
	require.Len(t, pkg.SimulationLogic.Variables, 3)
	assert.Equal(t, "振子质量", pkg.SimulationLogic.Variables[0].Label)
	assert.NotEmpty(t, pkg.SimulationLogic.RenderInstructions.Annotations)
	assert.False(t, pkg.SimulationLogic.LocalRecomputeAllowed == false)
	require.NotNil(t, pkg.InteractionPlan.Challenge)
	assert.NotEmpty(t, pkg.AssessmentTasks)
	assert.NotEmpty(t, pkg.RegenerationHints)
}
