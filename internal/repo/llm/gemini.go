package llm

import "github.com/beihai0xff/snowy/internal/pkg/config"

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
