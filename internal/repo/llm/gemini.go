package llm

import (
	"strings"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// geminiProvider 基于 Google Gemini API 的 LLM Provider。
type geminiProvider struct {
	unsupportedProvider

	cfg config.ModelProviderConfig
}

// NewGeminiProvider 创建 Gemini Provider。
func NewGeminiProvider(cfg config.ModelProviderConfig) Provider {
	return &geminiProvider{
		unsupportedProvider: unsupportedProvider{name: "gemini"},
		cfg:                 cfg,
	}
}

func (p *geminiProvider) ConfiguredModel() string {
	return p.cfg.EffectiveModel()
}

func (p *geminiProvider) ConfiguredBaseURL() string {
	return strings.TrimRight(p.cfg.EffectiveBaseURL(), "/")
}

func (p *geminiProvider) ConfiguredModelProvider() string {
	return strings.TrimSpace(p.cfg.ModelProvider)
}
