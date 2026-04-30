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
		"scene_type":"physics_projectile_2d",
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

func TestGenerateRenderPassesConfiguredProviderModel(t *testing.T) {
	provider := &renderFakeLLMProvider{model: "configured-render-model"}
	svc := &serviceImpl{primaryLLM: provider}

	_, err := svc.GenerateRender(context.Background(), &domain.SceneSpec{
		SceneType:  "physics_projectile_2d",
		Summary:    "projectile",
		RenderMode: domain.RenderModeHTMLIframe,
		DefaultProps: map[string]float64{
			"initialVelocity": 10,
		},
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

func TestForce3DUsesOptimizedPromptAndGenerationOptions(t *testing.T) {
	provider := &renderFakeLLMProvider{generateFn: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
		return nil, errors.New("stop after capture")
	}}
	svc := NewService(nil, WithLLMProviders(provider, nil))
	_, _ = svc.GenerateRender(context.Background(), &domain.SceneSpec{
		SceneType:    "physics_force_3d",
		Summary:      "force 3d",
		RenderMode:   domain.RenderModeHTMLIframe,
		DefaultProps: map[string]float64{"m": 2, "a": 3},
	}, string(domain.RenderModeHTMLIframe))

	if provider.captured == nil {
		t.Fatalf("expected provider request to be captured")
	}
	if provider.captured.Temperature != 0.1 {
		t.Fatalf("temperature = %v, want 0.1", provider.captured.Temperature)
	}
	if provider.captured.MaxTokens != 12288 {
		t.Fatalf("max tokens = %v, want 12288", provider.captured.MaxTokens)
	}
	if len(provider.captured.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(provider.captured.Messages))
	}
	systemPrompt := provider.captured.Messages[0].Content
	for _, want := range []string{"getContext('webgl')", "snowy:update-props", "snowy-preview ready", "view_dimension", "2D fallback", "perspective/mat4/camera/rotate", "不要为了压缩而省略"} {
		if !strings.Contains(systemPrompt, want) {
			t.Fatalf("system prompt missing %q", want)
		}
	}
	if strings.Contains(systemPrompt, "5000") {
		t.Fatalf("system prompt should not retain 5000 character cap")
	}
	userPrompt := provider.captured.Messages[1].Content
	for _, want := range []string{"不要省略 code_bundle", "getContext('webgl')", "snowy:update-props", "snowy-preview ready", "view_dimension", "2D fallback"} {
		if !strings.Contains(userPrompt, want) {
			t.Fatalf("user prompt missing %q", want)
		}
	}
}

func TestNonForce3DUsesDefaultGenerationOptions(t *testing.T) {
	provider := &renderFakeLLMProvider{generateFn: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
		return nil, errors.New("stop after capture")
	}}
	svc := NewService(nil, WithLLMProviders(provider, nil))
	_, _ = svc.GenerateRender(context.Background(), &domain.SceneSpec{
		SceneType:    "physics_projectile_2d",
		Summary:      "projectile",
		RenderMode:   domain.RenderModeHTMLIframe,
		DefaultProps: map[string]float64{"v0": 20},
	}, string(domain.RenderModeHTMLIframe))

	if provider.captured == nil {
		t.Fatalf("expected provider request to be captured")
	}
	if provider.captured.Temperature != 0.2 {
		t.Fatalf("temperature = %v, want 0.2", provider.captured.Temperature)
	}
	if provider.captured.MaxTokens != 8192 {
		t.Fatalf("max tokens = %v, want 8192", provider.captured.MaxTokens)
	}
	if strings.Contains(provider.captured.Messages[1].Content, "physics_force_3d 成功标准") {
		t.Fatalf("non-force scene should not include force 3d success standard in user prompt")
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
}

func TestForce3DTemplateContainsWebGLAndFallback(t *testing.T) {
	svc := NewService(nil)
	artifact, err := svc.GenerateRender(context.Background(), &domain.SceneSpec{
		SceneType:  "physics_force_3d",
		Title:      "牛顿第二定律 3D 受力模型",
		Summary:    "F=ma 3D model",
		RenderMode: domain.RenderModeHTMLIframe,
		DefaultProps: map[string]float64{
			"m": 2,
			"a": 3,
		},
	}, string(domain.RenderModeHTMLIframe))
	if err != nil {
		t.Fatalf("GenerateRender() error = %v", err)
	}
	if artifact.SceneType != "physics_force_3d" {
		t.Fatalf("scene type = %q", artifact.SceneType)
	}
	html := artifact.CodeBundle["index.html"]
	for _, want := range []string{"getContext('webgl')", "view_dimension", "snowy:update-props", "snowy-preview", "ready", "2D fallback", "mat4Perspective"} {
		if !strings.Contains(html, want) {
			t.Fatalf("index.html missing %q", want)
		}
	}
	if !containsString(artifact.RenderManifest.AllowedAPIs, "WebGLRenderingContext") {
		t.Fatalf("manifest allowed APIs missing WebGLRenderingContext: %#v", artifact.RenderManifest.AllowedAPIs)
	}
	if artifact.RenderManifest.InitialProps["view_dimension"] != 3 {
		t.Fatalf("manifest initial view_dimension = %v", artifact.RenderManifest.InitialProps["view_dimension"])
	}
}

func TestForce3DRejectsPure2DProviderArtifactAndFallsBack(t *testing.T) {
	provider := &renderFakeLLMProvider{generateFn: func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
		return &llm.Response{Content: `{
			"scene_type":"physics_force_3d",
			"render_mode":"html_iframe",
			"render_manifest":{"entry":"index.html","allowed_apis":["CanvasRenderingContext2D"]},
			"code_bundle":{"index.html":"<!doctype html><html><body><div id=\"snowy-preview-root\"></div><canvas id=\"c\"></canvas><script>window.addEventListener('message',function(event){if(event.data&&event.data.type==='snowy:update-props'){} }); parent.postMessage({source:'snowy-preview',type:'preview',status:'ready'}, '*');</script></body></html>"},
			"result_summary":"2d only"
		}`}, nil
	}}
	svc := NewService(nil, WithLLMProviders(provider, nil))
	artifact, err := svc.GenerateRender(context.Background(), &domain.SceneSpec{
		SceneType:    "physics_force_3d",
		Title:        "force 3d",
		Summary:      "force 3d",
		RenderMode:   domain.RenderModeHTMLIframe,
		DefaultProps: map[string]float64{"m": 2, "a": 3},
	}, string(domain.RenderModeHTMLIframe))
	if err != nil {
		t.Fatalf("GenerateRender() error = %v", err)
	}
	html := artifact.CodeBundle["index.html"]
	if !strings.Contains(html, "getContext('webgl')") {
		t.Fatalf("expected fallback WebGL template, got html prefix: %.160s", html)
	}
	if len(artifact.Warnings) == 0 || !strings.Contains(strings.Join(artifact.Warnings, " "), "missing native WebGL context") {
		t.Fatalf("expected warning with semantic validation failure, got %#v", artifact.Warnings)
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
