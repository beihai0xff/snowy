package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/beihai0xff/snowy/internal/modeling/physics/domain"
	"github.com/beihai0xff/snowy/internal/repo/llm"
)

func TestDecodeRenderArtifactAcceptsStringCodeBundle(t *testing.T) {
	artifact, err := decodeRenderArtifact(`{
		"scene_type":"physics_projectile_2d",
		"render_mode":"html_iframe",
		"render_manifest":{"entry":"index.html"},
		"code_bundle":"<!doctype html><html><body><div id=\"snowy-preview-root\"></div><script>parent.postMessage({source:'snowy-preview',type:'preview',status:'ready'}, '*');</script></body></html>",
		"result_summary":"ok"
	}`)
	if err != nil {
		t.Fatalf("decodeRenderArtifact() error = %v", err)
	}
	if artifact.CodeBundle["index.html"] == "" {
		t.Fatalf("expected string code_bundle normalized into index.html")
	}
}

func TestExtractJSONPayloadSkipsBracesInsideString(t *testing.T) {
	content := "prefix {not json} {\"code_bundle\":{\"index.html\":\"<script>const x = '{still string}';</script>\"}} suffix"
	payload := extractJSONPayload(content)
	want := "{\"code_bundle\":{\"index.html\":\"<script>const x = '{still string}';</script>\"}}"
	if payload != want {
		t.Fatalf("payload = %q, want %q", payload, want)
	}
}

type renderFakeLLMProvider struct {
	model      string
	captured   *llm.Request
	generateFn func(ctx context.Context, req *llm.Request) (*llm.Response, error)
}

func (f *renderFakeLLMProvider) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	f.captured = req
	if f.generateFn != nil {
		return f.generateFn(ctx, req)
	}
	return &llm.Response{Content: `{
		"scene_type":"biology_concept_flow",
		"render_mode":"html_iframe",
		"render_manifest":{"entry":"index.html"},
		"code_bundle":{"index.html":"<!doctype html><html><body><div id=\"snowy-preview-root\"></div><script>parent.postMessage({source:'snowy-preview',type:'preview',status:'ready'}, '*');</script></body></html>"},
		"result_summary":"ok"
	}`}, nil
}

func (f *renderFakeLLMProvider) GenerateStream(context.Context, *llm.Request, chan<- llm.StreamChunk) error {
	return nil
}

func (f *renderFakeLLMProvider) HealthCheck(context.Context) error { return nil }

func (f *renderFakeLLMProvider) EstimateCost(context.Context, *llm.Request) (*llm.Cost, error) {
	return &llm.Cost{}, nil
}

func (f *renderFakeLLMProvider) Name() string { return "render-fake" }

func (f *renderFakeLLMProvider) ConfiguredModel() string { return f.model }

func (f *renderFakeLLMProvider) ConfiguredBaseURL() string { return "" }

func (f *renderFakeLLMProvider) ConfiguredModelProvider() string { return "" }

func TestBiologyRenderPassesConfiguredProviderModel(t *testing.T) {
	provider := &renderFakeLLMProvider{model: "configured-render-model"}
	svc := &serviceImpl{llmChain: provider, codeValidator: nil}

	_, err := svc.GenerateRender(context.Background(), &domain.SceneSpec{
		SceneType:    "biology_concept_flow",
		Summary:      "biology",
		RenderMode:   domain.RenderModeHTMLIframe,
		DefaultProps: map[string]float64{"concept_count": 4},
	}, string(domain.RenderModeHTMLIframe))
	if err != nil {
		t.Fatalf("GenerateRender() error = %v", err)
	}
	if provider.captured == nil {
		t.Fatalf("expected provider request to be captured")
	}
	if provider.captured.Model != "configured-render-model" {
		t.Fatalf("request model = %q, want configured-render-model", provider.captured.Model)
	}
}

func TestPhysicsRenderReturnsNativeEngineArtifactWithoutLLM(t *testing.T) {
	provider := &renderFakeLLMProvider{generateFn: func(context.Context, *llm.Request) (*llm.Response, error) {
		t.Fatalf("physics render should not call LLM provider")
		return nil, nil
	}}
	svc := NewService(nil, WithLLMProvider(provider))
	artifact, err := svc.GenerateRender(context.Background(), &domain.SceneSpec{
		SceneType:    "physics_force_3d",
		Summary:      "force 3d",
		RenderMode:   domain.RenderModeHTMLIframe,
		DefaultProps: map[string]float64{"m": 2, "a": 3},
	}, string(domain.RenderModeHTMLIframe))
	if err != nil {
		t.Fatalf("GenerateRender() error = %v", err)
	}
	if provider.captured != nil {
		t.Fatalf("physics render captured LLM request: %#v", provider.captured)
	}
	if artifact.SceneType != "physics_force_3d" {
		t.Fatalf("scene type = %q", artifact.SceneType)
	}
	if artifact.RenderManifest.Framework != "snowy-native-physics-engine" {
		t.Fatalf("framework = %q", artifact.RenderManifest.Framework)
	}
	if artifact.RenderManifest.Sandbox != "react-native" {
		t.Fatalf("sandbox = %q", artifact.RenderManifest.Sandbox)
	}
	if !containsString(artifact.RenderManifest.Dependencies, "rapier3d") {
		t.Fatalf("dependencies missing rapier3d: %#v", artifact.RenderManifest.Dependencies)
	}
	if artifact.RenderManifest.InitialProps["view_dimension"] != 3 {
		t.Fatalf("manifest initial view_dimension = %v", artifact.RenderManifest.InitialProps["view_dimension"])
	}
	if _, ok := artifact.CodeBundle["README.md"]; !ok {
		t.Fatalf("native artifact should retain minimal README code_bundle")
	}
}

func TestProjectileRenderReturnsNativeEngineArtifactWithDefaults(t *testing.T) {
	svc := NewService(nil)
	artifact, err := svc.GenerateRender(context.Background(), &domain.SceneSpec{
		SceneType:    "physics_projectile_3d",
		Summary:      "projectile",
		RenderMode:   domain.RenderModeHTMLIframe,
		DefaultProps: map[string]float64{"v0": 30},
	}, string(domain.RenderModeHTMLIframe))
	if err != nil {
		t.Fatalf("GenerateRender() error = %v", err)
	}
	if artifact.RenderManifest.Framework != "snowy-native-physics-engine" {
		t.Fatalf("framework = %q", artifact.RenderManifest.Framework)
	}
	for _, key := range []string{"v0", "angle_deg", "t", "g", "view_dimension"} {
		if _, ok := artifact.RenderManifest.InitialProps[key]; !ok {
			t.Fatalf("initial props missing %q: %#v", key, artifact.RenderManifest.InitialProps)
		}
	}
}

func TestBiologyUsesOptimizedPromptAndGenerationOptions(t *testing.T) {
	provider := &renderFakeLLMProvider{generateFn: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
		return nil, errors.New("stop after capture")
	}}
	svc := NewService(nil, WithLLMProvider(provider))
	_, _ = svc.GenerateRender(context.Background(), &domain.SceneSpec{
		SceneType:    "biology_concept_flow",
		Summary:      "biology",
		RenderMode:   domain.RenderModeHTMLIframe,
		DefaultProps: map[string]float64{"concept_count": 4},
	}, string(domain.RenderModeHTMLIframe))

	if provider.captured == nil {
		t.Fatalf("expected provider request to be captured")
	}
	if provider.captured.Temperature != 0.15 {
		t.Fatalf("temperature = %v, want 0.15", provider.captured.Temperature)
	}
	if provider.captured.MaxTokens != llm.MaxTokens128K {
		t.Fatalf("max tokens = %v, want %v", provider.captured.MaxTokens, llm.MaxTokens128K)
	}
	systemPrompt := provider.captured.Messages[0].Content
	for _, want := range []string{"biology_*", "交互式科学可视化前端工程师", "animation_speed", "不限制 index.html/code_bundle 字符数"} {
		if !strings.Contains(systemPrompt, want) {
			t.Fatalf("system prompt missing %q", want)
		}
	}
	if strings.Contains(systemPrompt, "5000") {
		t.Fatalf("system prompt should not retain 5000 character cap")
	}
}

func TestNonPhysicsNonBiologyUsesDefaultGenerationOptions(t *testing.T) {
	provider := &renderFakeLLMProvider{generateFn: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
		return nil, errors.New("stop after capture")
	}}
	svc := NewService(nil, WithLLMProvider(provider))
	_, _ = svc.GenerateRender(context.Background(), &domain.SceneSpec{
		SceneType:    "custom_motion_2d",
		Summary:      "custom",
		RenderMode:   domain.RenderModeHTMLIframe,
		DefaultProps: map[string]float64{"v0": 20},
	}, string(domain.RenderModeHTMLIframe))

	if provider.captured == nil {
		t.Fatalf("expected provider request to be captured")
	}
	if provider.captured.Temperature != 0.2 {
		t.Fatalf("temperature = %v, want 0.2", provider.captured.Temperature)
	}
	if provider.captured.MaxTokens != llm.MaxTokens128K {
		t.Fatalf("max tokens = %v, want %v", provider.captured.MaxTokens, llm.MaxTokens128K)
	}
}

func TestAnalyzeNewtonSecondLawDefaultsToForce3D(t *testing.T) {
	svc := NewService(nil)
	model, err := svc.Analyze(context.Background(), "质量2kg的物体受到6N水平力，画出受力和加速度", "")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if model.SceneSpec == nil {
		t.Fatalf("expected scene spec")
	}
	if model.SceneSpec.SceneType != "physics_force_3d" {
		t.Fatalf("scene type = %q, want physics_force_3d", model.SceneSpec.SceneType)
	}
	if got := model.SceneSpec.DefaultProps["view_dimension"]; got != 3 {
		t.Fatalf("view_dimension = %v, want 3", got)
	}
	if !strings.Contains(strings.Join(model.Warnings, " "), "Rapier 3D") {
		t.Fatalf("expected Rapier warning, got %#v", model.Warnings)
	}
}

func TestAnalyzeNewTeachingScenes(t *testing.T) {
	tests := []struct {
		name      string
		question  string
		modelType domain.ModelType
		sceneType string
		props     []string
	}{
		{
			name:      "orbit",
			question:  "卫星绕地球运动，展示天体轨道和万有引力方向",
			modelType: domain.ModelTwoBodyMotion,
			sceneType: "physics_orbit_3d",
			props:     []string{"central_mass", "satellite_mass", "orbit_radius", "tangential_speed", "eccentricity", "gravitational_strength", "view_dimension"},
		},
		{
			name:      "spring",
			question:  "弹簧振子简谐运动，展示回复力和能量变化",
			modelType: domain.ModelSpringOscillator,
			sceneType: "physics_spring_3d",
			props:     []string{"k", "m", "x", "damping", "view_dimension"},
		},
		{
			name:      "collision",
			question:  "两个小球弹性碰撞，展示动量和能量变化",
			modelType: domain.ModelCollisionMotion,
			sceneType: "physics_collision_3d",
			props:     []string{"m1", "m2", "v1", "v2", "restitution", "view_dimension"},
		},
	}

	svc := NewService(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := svc.Analyze(context.Background(), tt.question, "")
			if err != nil {
				t.Fatalf("Analyze() error = %v", err)
			}
			if model.ModelType != tt.modelType {
				t.Fatalf("model type = %q, want %q", model.ModelType, tt.modelType)
			}
			if model.SceneSpec == nil {
				t.Fatalf("expected scene spec")
			}
			if model.SceneSpec.SceneType != tt.sceneType {
				t.Fatalf("scene type = %q, want %q", model.SceneSpec.SceneType, tt.sceneType)
			}
			for _, key := range tt.props {
				if _, ok := model.SceneSpec.DefaultProps[key]; !ok {
					t.Fatalf("default props missing %q: %#v", key, model.SceneSpec.DefaultProps)
				}
			}
		})
	}
}

func TestAnalyzeOrbitQuestionDoesNotTreat3DAsCondition(t *testing.T) {
	svc := NewService(nil)
	model, err := svc.Analyze(context.Background(), "卫星绕地球做轨道运动，展示天体运动 3D 模型", "")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if model.ModelType != domain.ModelTwoBodyMotion {
		t.Fatalf("model type = %q, want %q", model.ModelType, domain.ModelTwoBodyMotion)
	}
	if model.SceneSpec == nil || model.SceneSpec.SceneType != "physics_orbit_3d" {
		t.Fatalf("scene spec = %#v", model.SceneSpec)
	}
	for _, condition := range model.Conditions {
		if condition.Name == "v0" || condition.Unit == "D" {
			t.Fatalf("3D view hint leaked into extracted conditions: %#v", model.Conditions)
		}
	}
	if !strings.Contains(model.ResultSummary, "gravity_force=") {
		t.Fatalf("summary should use normalized orbit metrics, got %q", model.ResultSummary)
	}
	if strings.Contains(model.ResultSummary, "198611753234863") {
		t.Fatalf("summary still contains unreadable SI astronomy force: %q", model.ResultSummary)
	}
}

func TestNewPhysicsScenesRenderNativeArtifactWithoutLLM(t *testing.T) {
	provider := &renderFakeLLMProvider{generateFn: func(context.Context, *llm.Request) (*llm.Response, error) {
		t.Fatalf("physics render should not call LLM provider")
		return nil, nil
	}}
	svc := NewService(nil, WithLLMProvider(provider))

	tests := []struct {
		sceneType string
		props     []string
	}{
		{sceneType: "physics_orbit_3d", props: []string{"central_mass", "satellite_mass", "orbit_radius", "tangential_speed", "gravitational_strength", "view_dimension"}},
		{sceneType: "physics_spring_3d", props: []string{"k", "m", "x", "damping", "view_dimension"}},
		{sceneType: "physics_collision_3d", props: []string{"m1", "m2", "v1", "v2", "restitution", "view_dimension"}},
	}

	for _, tt := range tests {
		t.Run(tt.sceneType, func(t *testing.T) {
			artifact, err := svc.GenerateRender(context.Background(), &domain.SceneSpec{
				SceneType:  tt.sceneType,
				Title:      tt.sceneType,
				RenderMode: domain.RenderModeHTMLIframe,
			}, string(domain.RenderModeHTMLIframe))
			if err != nil {
				t.Fatalf("GenerateRender() error = %v", err)
			}
			if provider.captured != nil {
				t.Fatalf("physics render captured LLM request: %#v", provider.captured)
			}
			if artifact.RenderManifest.Framework != "snowy-native-physics-engine" {
				t.Fatalf("framework = %q", artifact.RenderManifest.Framework)
			}
			for _, key := range tt.props {
				if _, ok := artifact.RenderManifest.InitialProps[key]; !ok {
					t.Fatalf("initial props missing %q: %#v", key, artifact.RenderManifest.InitialProps)
				}
			}
		})
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
