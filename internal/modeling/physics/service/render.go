package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	scene := *sceneSpec
	scene.RenderMode = mode
	if scene.DefaultProps == nil {
		scene.DefaultProps = map[string]float64{}
	}

	var attemptErrors []string

	for _, provider := range []llm.Provider{s.primaryLLM, s.fallbackLLM} {
		artifact, err := s.tryGenerateWithProvider(ctx, provider, &scene, mode)
		if err == nil && artifact != nil {
			return artifact, nil
		}
		if err != nil {
			attemptErrors = append(attemptErrors, err.Error())
		}
	}

	artifact, err := s.generateTemplateArtifact(&scene, mode)
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

func (s *serviceImpl) tryGenerateWithProvider(
	ctx context.Context,
	provider llm.Provider,
	sceneSpec *domain.SceneSpec,
	mode domain.RenderMode,
) (*domain.RenderArtifact, error) {
	if provider == nil {
		return nil, errors.New("llm provider is nil")
	}

	temperature, maxTokens := renderGenerationOptions(sceneSpec)
	request := &llm.Request{
		Model: providerModel(provider),
		Messages: []llm.Message{
			{Role: "system", Content: renderSystemPrompt()},
			{Role: "user", Content: buildRenderUserPrompt(sceneSpec, mode)},
		},
		Temperature: temperature,
		MaxTokens:   maxTokens,
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
	return strings.TrimSpace(`你是一个前端代码生成器。不要展开推理，直接输出最终 JSON，不要 markdown，不要解释文字。
目标：根据 scene_spec 生成可放入 iframe srcDoc 运行的原生 HTML/CSS/JavaScript 预览，必须能在无网络、iframe sandbox 中运行。优先保证 JSON 合法、code_bundle 完整、代码可执行；保持代码紧凑，但不要为了压缩而省略必要实现或输出占位符。

必须严格输出这个 JSON schema：
{
  "scene_type": "physics_force_3d",
  "render_mode": "html_iframe",
  "render_manifest": {
    "entry": "index.html",
    "framework": "vanilla",
    "sandbox": "iframe",
    "render_mode": "html_iframe",
    "mount_selector": "#snowy-preview-root",
    "dependencies": ["native-html", "webgl", "canvas"],
    "allowed_apis": ["requestAnimationFrame", "setTimeout", "postMessage", "CanvasRenderingContext2D", "WebGLRenderingContext", "WebGL2RenderingContext"],
    "blocked_apis": ["fetch", "XMLHttpRequest", "localStorage", "sessionStorage", "document.cookie", "WebSocket", "navigator.sendBeacon"],
    "initial_props": {"m":2,"a":3,"view_dimension":3,"camera_yaw":0.55,"camera_pitch":0.42}
  },
  "code_bundle": {"index.html":"<!doctype html>..."},
  "result_summary": "..."
}

硬性要求：
1. code_bundle 必须是 JSON object，至少包含 index.html；不要把 code_bundle 输出成字符串；不要输出 markdown、解释文字或省略号占位符。
2. index.html 必须是完整 HTML，包含 id="snowy-preview-root" 的根节点；所有 CSS/JS 内联，禁止外链脚本、CDN、Three.js、动态依赖和网络请求；physics_force_3d 优先只生成单文件 index.html。
3. 禁止在 index.html 的代码、注释、字符串、HTML 属性中出现这些危险片段：fetch(、XMLHttpRequest、localStorage、sessionStorage、indexedDB、document.cookie、WebSocket、navigator.sendBeacon、<script src=、import(。
4. JS 必须监听 window message：event.data.type === 'snowy:update-props' 时合并 props 并重绘；支持数值 prop view_dimension=3 或 2 切换视图。
5. 渲染 ready 后必须调用 parent.postMessage({source:'snowy-preview',type:'preview',status:'ready'}, '*')；异常时发送 type:'error'。
6. 整体视觉必须像“炫酷教学演示页面”：暗色舞台、渐变/霓虹高光、动态图例、参数 HUD、阶段说明面板、发光箭头或粒子流、平滑动画；不要只画静态黑白示意图。
7. 物理场景风格：physics_force_3d 默认原生 WebGL 3D，包含动态网格、发光力矢量、加速度矢量、质量 HUD、可拖拽/2D切换；physics_projectile_* 强调轨迹残影、关键点、速度/时间面板；physics_motion_* 强调时间轴、状态卡片和运动方向。
8. scene_type=physics_force_3d 时必须使用原生 WebGLRenderingContext 或 WebGL2RenderingContext 绘制默认 3D，不要只输出 2D Canvas。
9. physics_force_3d 推荐最稳实现路径：一个 WebGL canvas 绘制 3D，一个 CanvasRenderingContext2D overlay 画文字标签；使用简单 shader、perspective/mat4/camera/rotate 矩阵或等价旋转投影；用 lines/triangles 画地面 grid、XYZ axis、box、force vector、acceleration vector，不要追求复杂引擎式效果。
10. physics_force_3d 同一个 index.html 内必须带 3D / 2D 按钮和 Canvas 2D fallback；只有 WebGL 不可用或 view_dimension=2 时才切换 2D fallback。
11. biology_* 场景可以使用 Canvas 2D 或 WebGL；必须包含粒子/流动路径、阶段切换、概念标签、过程箭头、解释面板、播放/暂停或自动动画。biology_photosynthesis_3d 要表现叶绿体、光子、CO₂/H₂O 输入、O₂/糖输出；biology_cell_process_3d 要表现细胞膜/细胞器/物质运输；biology_concept_flow 要表现动态概念关系网络。
12. 输出前自检：JSON 必须包含 code_bundle.index.html；physics_force_3d 的 index.html 中必须能找到 getContext('webgl') 或 getContext("webgl")、snowy:update-props、snowy-preview ready、view_dimension、2D fallback、perspective/mat4/camera/rotate 之一；biology_* 的 index.html 中必须能找到 snowy:update-props、snowy-preview ready、CanvasRenderingContext2D 或 getContext('2d')、particle/flow/stage/label 等可视化语义。
13. JSON 字符串中的换行和引号必须合法转义；如果代码较长，继续完整输出，不要截断 code_bundle。`)
}

func buildRenderUserPrompt(sceneSpec *domain.SceneSpec, mode domain.RenderMode) string {
	payload, _ := json.Marshal(sceneSpec)
	prompt := fmt.Sprintf("scene_spec=%s\nrender_mode=%s\n只输出 JSON，不要 markdown；不要省略 code_bundle，不要用占位符。", string(payload), mode)
	if sceneSpec != nil && sceneSpec.SceneType == "physics_force_3d" {
		prompt += "\n\nphysics_force_3d 成功标准：code_bundle.index.html 必须是完整单文件 HTML；必须初始化 WebGL，例如 getContext('webgl')；必须监听 snowy:update-props；必须发送 snowy-preview ready；必须支持 view_dimension=3/2；必须包含 2D fallback；必须包含 perspective/mat4/camera/rotate 等 3D 视角语义。代码较长也要完整输出。"
	}
	return prompt
}

func renderGenerationOptions(sceneSpec *domain.SceneSpec) (float64, int) {
	if sceneSpec != nil && sceneSpec.SceneType == "physics_force_3d" {
		return 0.1, 12288
	}
	if sceneSpec != nil && strings.HasPrefix(sceneSpec.SceneType, "biology_") {
		return 0.15, 12288
	}
	return 0.2, 8192
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

	return fmt.Errorf("unsupported code_bundle shape")
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
	for key, value := range input {
		cloned[key] = value
	}

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
		manifest.AllowedAPIs = []string{"requestAnimationFrame", "setTimeout", "postMessage", "CanvasRenderingContext2D", "WebGLRenderingContext", "WebGL2RenderingContext"}
	}
	if len(manifest.BlockedAPIs) == 0 {
		manifest.BlockedAPIs = []string{"fetch", "XMLHttpRequest", "localStorage", "sessionStorage", "indexedDB", "document.cookie", "WebSocket", "navigator.sendBeacon"}
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
		{strings.Contains(joined, "getcontext('webgl") || strings.Contains(joined, "getcontext(\"webgl") || strings.Contains(joined, "webgl2"), "missing native WebGL context"},
		{strings.Contains(joined, "snowy:update-props"), "missing update-props listener"},
		{strings.Contains(joined, "snowy-preview") && strings.Contains(joined, "ready"), "missing ready postMessage handshake"},
		{strings.Contains(joined, "view_dimension") || (strings.Contains(joined, "3d") && strings.Contains(joined, "2d")), "missing 3D/2D view switch"},
		{containsAny(joined, "perspective", "mat4", "projection", "camera", "rotate"), "missing 3D camera/projection semantics"},
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
