//nolint:cyclop // Render prompting keeps long model contracts and JSON extraction logic explicit.
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
	"github.com/beihai0xff/snowy/internal/prompt"
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

	artifact, err := s.tryGenerateWithProvider(ctx, s.llmChain, &scene, mode)
	if err != nil {
		return nil, fmt.Errorf("llm render generation failed: %w", err)
	}

	if artifact == nil {
		return nil, errors.New("llm render generation failed: empty artifact")
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
			{Role: "system", Content: prompt.RenderSystem()},
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

	if artifact.SceneType == scenePhysicsForce3D {
		if err := validateForce3DArtifact(artifact.CodeBundle); err != nil {
			return err
		}
	}

	return nil
}

func buildRenderUserPrompt(sceneSpec *domain.SceneSpec, mode domain.RenderMode) string {
	return prompt.RenderUser(prompt.RenderInput{
		SceneSpec: sceneSpec,
		Mode:      string(mode),
		Biology:   sceneSpec != nil && strings.HasPrefix(sceneSpec.SceneType, "biology_"),
	})
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

func (r *rawRenderArtifact) toDomain() *domain.RenderArtifact {
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
	case domain.RenderModeHTMLIframe:
		return domain.RenderModeHTMLIframe
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
