package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"

	"github.com/beihai0xff/snowy/internal/modeling/physics/domain"
	"github.com/beihai0xff/snowy/internal/repo/llm"
)

func (s *serviceImpl) GenerateRender(
	ctx context.Context,
	sceneSpec *domain.SceneSpec,
	renderMode string,
) (*domain.RenderArtifact, error) {
	if sceneSpec == nil || strings.TrimSpace(sceneSpec.SceneType) == "" {
		return nil, errors.New("scene spec is required")
	}

	mode := normalizeRenderMode(renderMode, sceneSpec.RenderMode)
	if isPhysicsScene(sceneSpec.SceneType) {
		return s.generateNativePhysicsArtifact(sceneSpec, mode)
	}

	return s.generateLLMRenderArtifact(ctx, sceneSpec, mode)
}

func (s *serviceImpl) generateLLMRenderArtifact(
	ctx context.Context,
	sceneSpec *domain.SceneSpec,
	mode domain.RenderMode,
) (*domain.RenderArtifact, error) {
	scene := *sceneSpec

	scene.RenderMode = mode
	if scene.DefaultProps == nil {
		scene.DefaultProps = map[string]float64{}
	}

	var attemptErrors []string

	artifact, err := s.tryGenerateWithProvider(ctx, s.llmChain, &scene, mode)
	if err == nil && artifact != nil {
		return artifact, nil
	}
	if err != nil {
		attemptErrors = append(attemptErrors, err.Error())
	}

	artifact, err = s.generateTemplateArtifact(&scene, mode)
	if err != nil {
		attemptErrors = append(attemptErrors, err.Error())

		return nil, fmt.Errorf("render generation failed: %s", strings.Join(attemptErrors, " | "))
	}

	if len(attemptErrors) > 0 {
		artifact.Warnings = append(
			artifact.Warnings,
			"大模型生成未通过，已回退到本地模板生成代码："+strings.Join(attemptErrors, " | "),
		)
	} else {
		artifact.Warnings = append(artifact.Warnings, "当前未配置可用大模型提供方，已回退到本地模板生成代码")
	}

	return artifact, nil
}

func isPhysicsScene(sceneType string) bool {
	return strings.HasPrefix(strings.TrimSpace(sceneType), "physics_")
}

func (s *serviceImpl) tryGenerateWithProvider(
	ctx context.Context,
	provider llm.Provider,
	sceneSpec *domain.SceneSpec,
	mode domain.RenderMode,
) (*domain.RenderArtifact, error) {
	if provider == nil {
		return nil, errors.New("llm provider is nil")
	}

	temperature := renderGenerationTemperature(sceneSpec)
	request := &llm.Request{
		Model: providerModel(provider),
		Messages: []llm.Message{
			{Role: "system", Content: renderSystemPrompt()},
			{Role: "user", Content: buildRenderUserPrompt(sceneSpec, mode)},
		},
		Temperature: temperature,
		MaxTokens:   llm.MaxTokens128K,
	}

	response, err := provider.Generate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("provider %s generate failed: %w", provider.Name(), err)
	}

	artifact, err := decodeRenderArtifact(response.Content)
	if err != nil {
		return nil, fmt.Errorf("provider %s decode artifact failed: %w", provider.Name(), err)
	}

	if artifact.SceneType == "" {
		artifact.SceneType = sceneSpec.SceneType
	}

	if artifact.RenderMode == "" {
		artifact.RenderMode = mode
	}

	if artifact.ResultSummary == "" {
		artifact.ResultSummary = sceneSpec.Summary
	}

	if artifact.RenderManifest == nil {
		artifact.RenderManifest = &domain.RenderManifest{}
	}

	applyManifestDefaults(artifact.RenderManifest, sceneSpec, mode)

	if len(artifact.CodeBundle) == 0 {
		return nil, errors.New("provider returned empty code bundle")
	}

	if err := s.validateArtifact(artifact); err != nil {
		return nil, err
	}

	return artifact, nil
}

func (s *serviceImpl) validateArtifact(artifact *domain.RenderArtifact) error {
	if artifact == nil {
		return errors.New("artifact is nil")
	}

	if artifact.RenderManifest == nil {
		return errors.New("render manifest is nil")
	}

	if s.codeValidator != nil {
		if err := s.codeValidator.Validate(artifact.CodeBundle); err != nil {
			return err
		}
	}

	if artifact.SceneType == "physics_force_3d" {
		if err := validateForce3DArtifact(artifact.CodeBundle); err != nil {
			return err
		}
	}

	return nil
}

func renderSystemPrompt() string {
	return strings.TrimSpace(
		`你是一名专业的交互式科学可视化前端工程师。任务是根据 scene_spec 生成一个可在浏览器 iframe srcDoc 中独立运行的原生 HTML/CSS/JavaScript 教学演示页面。不要展开推理，不要输出 markdown，不要输出解释文字；最终只输出符合约定的 JSON。

总体目标：
- 生成离线可运行、无需网络、无需外部依赖、适合 iframe sandbox 的单文件交互式演示。
- 优先保证 JSON 合法、code_bundle 完整、代码可执行和教学信息准确；不限制 index.html/code_bundle 字符数，不要为了压缩而省略必要实现或使用占位符。
- 页面应具备专业的科普/课堂演示质感：清晰结构、动态视觉、关键标签、参数反馈、状态提示和可理解的过程呈现。

必须严格输出这个 JSON schema：
{
  "scene_type": "biology_concept_flow",
  "render_mode": "html_iframe",
  "render_manifest": {
    "entry": "index.html",
    "framework": "vanilla",
    "sandbox": "iframe",
    "render_mode": "html_iframe",
    "mount_selector": "#snowy-preview-root",
    "dependencies": ["native-html", "canvas"],
    "allowed_apis": ["requestAnimationFrame", "setTimeout", "postMessage", "CanvasRenderingContext2D"],
    "blocked_apis": ["fetch", "XMLHttpRequest", "localStorage", "sessionStorage", "document.cookie", "WebSocket", "navigator.sendBeacon"],
    "initial_props": {"concept_count":4,"relation_count":3,"animation_speed":1}
  },
  "code_bundle": {"index.html":"<!doctype html>..."},
  "result_summary": "..."
}

硬性要求：
1. code_bundle 必须是 JSON object，至少包含 index.html；不要把 code_bundle 输出成字符串；不要输出 markdown、解释文字、省略号或“待实现”占位符。
2. index.html 必须是完整 HTML，包含 id="snowy-preview-root" 的根节点；所有 CSS/JS 内联；禁止外链脚本、CDN、动态依赖和任何网络访问。
3. 禁止在 index.html 的代码、注释、字符串、HTML 属性中出现这些危险片段：fetch(、XMLHttpRequest、localStorage、sessionStorage、indexedDB、document.cookie、WebSocket、navigator.sendBeacon、<script src=、import(。
4. JS 必须监听 window message：event.data.type === 'snowy:update-props' 时合并 props 并重绘；支持数值 prop view_dimension=3 或 2 切换视图；支持 animation_speed 调整动画倍率。
5. 渲染 ready 后必须调用 parent.postMessage({source:'snowy-preview',type:'preview',status:'ready'}, '*')；异常时发送 parent.postMessage({source:'snowy-preview',type:'preview',status:'error',message:String(error)}, '*')。
6. 视觉要求：暗色或高对比舞台、渐变/霓虹高光、动态图例、参数 HUD、阶段说明面板、发光箭头或粒子流、平滑动画；避免静态黑白示意图或信息密度过低的画面。
7. physics_* 场景由宿主应用的本地物理引擎承载；不要为 physics_* 生成可执行前端代码。本生成链路主要服务 biology_* 等非物理可视化场景。
8. biology_* 场景可以使用 Canvas 2D 或 WebGL；必须包含粒子/流动路径、阶段切换、概念标签、过程箭头、解释面板、播放/暂停或自动动画。biology_photosynthesis_3d 要表现叶绿体、光子、CO₂/H₂O 输入、O₂/糖输出；biology_cell_process_3d 要表现细胞膜/细胞器/物质运输；biology_concept_flow 要表现动态概念关系网络。
9. 输出前自检：JSON 必须包含 code_bundle.index.html；biology_* 的 index.html 中必须能找到 snowy:update-props、snowy-preview ready、CanvasRenderingContext2D 或 getContext('2d')、particle/flow/stage/label 等可视化语义。
10. JSON 字符串中的换行和引号必须合法转义；代码包长度不设上限，如果代码较长，继续完整输出，不要截断 code_bundle。`,
	)
}

func buildRenderUserPrompt(sceneSpec *domain.SceneSpec, mode domain.RenderMode) string {
	payload, _ := json.Marshal(sceneSpec)

	prompt := fmt.Sprintf(
		"scene_spec=%s\nrender_mode=%s\n只输出 JSON，不要 markdown；不要省略 code_bundle，不要用占位符；代码包长度不设上限，必须完整输出。",
		string(payload),
		mode,
	)
	if sceneSpec != nil && strings.HasPrefix(sceneSpec.SceneType, "biology_") {
		prompt += "\n\nbiology_* 成功标准：code_bundle.index.html 必须是完整单文件 HTML；必须监听 snowy:update-props；必须发送 snowy-preview ready；必须包含 CanvasRenderingContext2D 或 getContext('2d')；必须包含 particle/flow/stage/label 等可视化语义；代码包长度不设上限，代码较长也要完整输出。"
	}

	return prompt
}

func renderGenerationTemperature(sceneSpec *domain.SceneSpec) float64 {
	if sceneSpec != nil && strings.HasPrefix(sceneSpec.SceneType, "biology_") {
		return 0.15
	}

	return 0.2
}

func decodeRenderArtifact(content string) (*domain.RenderArtifact, error) {
	payload := extractJSONPayload(content)
	if payload == "" {
		return nil, errors.New("empty JSON payload")
	}

	var raw rawRenderArtifact
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return nil, err
	}

	artifact := raw.toDomain()
	if len(artifact.CodeBundle) == 0 && strings.TrimSpace(raw.CodeBundleString) != "" {
		artifact.CodeBundle = map[string]string{"index.html": raw.CodeBundleString}
	}

	return artifact, nil
}

type rawRenderArtifact struct {
	SceneType        string                 `json:"scene_type"`
	RenderMode       domain.RenderMode      `json:"render_mode"`
	RenderManifest   *domain.RenderManifest `json:"render_manifest"`
	CodeBundle       map[string]string      `json:"code_bundle"`
	CodeBundleString string                 `json:"-"`
	ResultSummary    string                 `json:"result_summary"`
	Warnings         []string               `json:"warnings,omitempty"`
}

func (r *rawRenderArtifact) UnmarshalJSON(data []byte) error {
	type alias rawRenderArtifact

	var aux struct {
		*alias

		CodeBundle json.RawMessage `json:"code_bundle"`
	}

	aux.alias = (*alias)(r)
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	trimmed := strings.TrimSpace(string(aux.CodeBundle))
	if trimmed == "" || trimmed == "null" {
		return nil
	}

	if strings.HasPrefix(trimmed, "{") {
		if err := json.Unmarshal(aux.CodeBundle, &r.CodeBundle); err != nil {
			return err
		}

		return nil
	}

	if strings.HasPrefix(trimmed, "\"") {
		return json.Unmarshal(aux.CodeBundle, &r.CodeBundleString)
	}

	return errors.New("unsupported code_bundle shape")
}

func (r rawRenderArtifact) toDomain() *domain.RenderArtifact {
	return &domain.RenderArtifact{
		SceneType:      r.SceneType,
		RenderMode:     r.RenderMode,
		RenderManifest: r.RenderManifest,
		CodeBundle:     cloneStringMap(r.CodeBundle),
		ResultSummary:  r.ResultSummary,
		Warnings:       append([]string(nil), r.Warnings...),
	}
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(input))
	maps.Copy(cloned, input)

	return cloned
}

func extractJSONPayload(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}

	trimmed = stripMarkdownFence(trimmed)
	if json.Valid([]byte(trimmed)) {
		return trimmed
	}

	if payload := firstJSONObject(trimmed); payload != "" {
		return payload
	}

	return trimmed
}

var markdownJSONFencePattern = regexp.MustCompile("(?is)^```(?:json)?\\s*(.*?)\\s*```$")

func stripMarkdownFence(content string) string {
	trimmed := strings.TrimSpace(content)

	matches := markdownJSONFencePattern.FindStringSubmatch(trimmed)
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}

	return trimmed
}

func firstJSONObject(content string) string {
	for start := strings.Index(content, "{"); start >= 0; {
		if payload := jsonObjectFrom(content, start); payload != "" {
			return payload
		}

		next := strings.Index(content[start+1:], "{")
		if next < 0 {
			break
		}

		start += next + 1
	}

	return ""
}

func jsonObjectFrom(content string, start int) string {
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(content); i++ {
		ch := content[i]

		if inString {
			if escaped {
				escaped = false

				continue
			}

			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}

			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				candidate := content[start : i+1]
				if json.Valid([]byte(candidate)) {
					return candidate
				}

				return ""
			}
		}
	}

	return ""
}

func normalizeRenderMode(input string, fallback domain.RenderMode) domain.RenderMode {
	if strings.TrimSpace(input) == "" {
		if fallback != "" {
			return fallback
		}

		return domain.RenderModeHTMLIframe
	}

	switch domain.RenderMode(strings.TrimSpace(input)) {
	case domain.RenderModeReactIframe:
		return domain.RenderModeReactIframe
	default:
		return domain.RenderModeHTMLIframe
	}
}

func applyManifestDefaults(manifest *domain.RenderManifest, sceneSpec *domain.SceneSpec, mode domain.RenderMode) {
	if manifest.Entry == "" {
		manifest.Entry = "index.html"
	}

	if manifest.Framework == "" {
		manifest.Framework = "vanilla"
	}

	if manifest.Sandbox == "" {
		manifest.Sandbox = "iframe"
	}

	if manifest.RenderMode == "" {
		manifest.RenderMode = mode
	}

	if manifest.MountSelector == "" {
		manifest.MountSelector = "#snowy-preview-root"
	}

	if len(manifest.AllowedAPIs) == 0 {
		manifest.AllowedAPIs = []string{
			"requestAnimationFrame",
			"setTimeout",
			"postMessage",
			"CanvasRenderingContext2D",
			"WebGLRenderingContext",
			"WebGL2RenderingContext",
		}
	}

	if len(manifest.BlockedAPIs) == 0 {
		manifest.BlockedAPIs = []string{
			"fetch",
			"XMLHttpRequest",
			"localStorage",
			"sessionStorage",
			"indexedDB",
			"document.cookie",
			"WebSocket",
			"navigator.sendBeacon",
		}
	}

	if len(manifest.InitialProps) == 0 {
		manifest.InitialProps = cloneNumberMap(sceneSpec.DefaultProps)
	}
}

func validateForce3DArtifact(bundle map[string]string) error {
	joined := strings.ToLower(strings.Join(bundleValues(bundle), "\n"))

	checks := []struct {
		ok      bool
		message string
	}{
		{
			strings.Contains(joined, "getcontext('webgl") || strings.Contains(joined, "getcontext(\"webgl") ||
				strings.Contains(joined, "webgl2"),
			"missing native WebGL context",
		},
		{strings.Contains(joined, "snowy:update-props"), "missing update-props listener"},
		{
			strings.Contains(joined, "snowy-preview") && strings.Contains(joined, "ready"),
			"missing ready postMessage handshake",
		},
		{
			strings.Contains(joined, "view_dimension") ||
				(strings.Contains(joined, "3d") && strings.Contains(joined, "2d")),
			"missing 3D/2D view switch",
		},
		{
			containsAny(joined, "perspective", "mat4", "projection", "camera", "rotate"),
			"missing 3D camera/projection semantics",
		},
	}
	for _, check := range checks {
		if !check.ok {
			return fmt.Errorf("physics_force_3d artifact invalid: %s", check.message)
		}
	}

	return nil
}

func bundleValues(bundle map[string]string) []string {
	values := make([]string, 0, len(bundle))
	for _, value := range bundle {
		values = append(values, value)
	}

	return values
}

func providerModel(provider llm.Provider) string {
	if configured, ok := provider.(llm.ConfiguredProvider); ok {
		return configured.ConfiguredModel()
	}

	return ""
}
