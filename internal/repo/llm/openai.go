package llm

import "github.com/beihai0xff/snowy/internal/pkg/config"

// openaiProvider 基于 OpenAI API 的 LLM Provider。
// 生产环境将通过 Eino ChatModel 封装。
type openaiProvider struct {
	unsupportedProvider

	cfg config.ModelProviderConfig
}

// NewOpenAIProvider 创建 OpenAI Provider。
func NewOpenAIProvider(cfg config.ModelProviderConfig) Provider {
	return &openaiProvider{
		unsupportedProvider: unsupportedProvider{name: "openai"},
		cfg:                 cfg,
	}
}
