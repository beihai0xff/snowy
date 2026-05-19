package prompt

import (
	"strings"
	"testing"
	"time"
)

func TestCatalogVersionsAreStable(t *testing.T) {
	profiles := DefaultProfiles(time.Date(2026, 5, 19, 0, 0, 0, 0, time.UTC))
	if len(profiles) == 0 {
		t.Fatal("profiles are empty")
	}

	for _, profile := range profiles {
		if strings.TrimSpace(profile.ID) == "" || strings.TrimSpace(profile.Version) == "" {
			t.Fatalf("profile missing id/version: %#v", profile)
		}

		if profile.UpdatedAt.IsZero() {
			t.Fatalf("profile missing updated_at: %#v", profile)
		}
	}
}

func TestSystemPromptsContainSafetyContracts(t *testing.T) {
	cases := map[string]string{
		"search":     KnowledgeAnswerSystem(),
		"modeling":   ModelingCompileSystem(),
		"render":     RenderSystem(),
		"classifier": RegenerateClassifierSystem(),
	}

	for name, content := range cases {
		if !strings.Contains(content, "不展示隐藏") {
			t.Fatalf("%s prompt missing no-hidden-reasoning constraint", name)
		}
	}

	for _, content := range []string{ModelingCompileSystem(), RenderSystem(), RegenerateClassifierSystem()} {
		if !strings.Contains(content, "JSON") {
			t.Fatalf("json prompt missing JSON-only constraint: %s", content)
		}
	}

	if !strings.Contains(RenderSystem(), "blocked_apis") || !strings.Contains(RenderSystem(), "postMessage") {
		t.Fatal("render prompt missing sandbox protocol constraints")
	}
}

func TestUserPromptRenderers(t *testing.T) {
	searchPrompt := KnowledgeAnswerUser(KnowledgeAnswerInput{
		Date:     time.Date(2026, 5, 19, 0, 0, 0, 0, time.UTC),
		Question: "牛顿第二定律是什么？",
		Subject:  "physics",
		Grade:    "high-school",
		Intent:   "explain",
		Entities: []string{"牛顿", "加速度"},
	})
	for _, want := range []string{"2026-05-19", "牛顿第二定律", "不要输出 JSON", "不要伪造引用"} {
		if !strings.Contains(searchPrompt, want) {
			t.Fatalf("search prompt missing %q: %s", want, searchPrompt)
		}
	}

	compilePrompt := ModelingCompileUser(ModelingCompileInput{Domain: "chemistry", Message: "配平 H2 + O2"})
	for _, want := range []string{"physics|biology|chemistry", "chemistry", "配平 H2 + O2"} {
		if !strings.Contains(compilePrompt, want) {
			t.Fatalf("compile prompt missing %q: %s", want, compilePrompt)
		}
	}

	renderPrompt := RenderUser(RenderInput{SceneSpec: map[string]any{"scene_type": "biology_concept_flow"}, Mode: "html_iframe", Biology: true})
	for _, want := range []string{"scene_spec=", "html_iframe", "biology_* 成功标准"} {
		if !strings.Contains(renderPrompt, want) {
			t.Fatalf("render prompt missing %q: %s", want, renderPrompt)
		}
	}
}
