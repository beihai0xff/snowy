package llm

import (
	"context"
	"fmt"

	"github.com/beihai0xff/snowy/internal/config"
)

// openaiProvider 基于 OpenAI API 的 LLM Provider。
// 生产环境将通过 Eino ChatModel 封装。
type openaiProvider struct {
	cfg config.ModelProviderConfig
}

// NewOpenAIProvider 创建 OpenAI Provider。
func NewOpenAIProvider(cfg config.ModelProviderConfig) Provider {
	return &openaiProvider{cfg: cfg}
}

func (p *openaiProvider) Name() string { return "openai" }

func (p *openaiProvider) Generate(ctx context.Context, req *Request) (*Response, error) {
	// TODO: 通过 Eino openai.ChatModel 实现
	return nil, fmt.Errorf("openai provider: not implemented")
}

func (p *openaiProvider) GenerateStream(ctx context.Context, req *Request, chunks chan<- StreamChunk) error {
	// TODO: 通过 Eino openai.ChatModel streaming 实现
	return fmt.Errorf("openai provider stream: not implemented")
}

func (p *openaiProvider) HealthCheck(ctx context.Context) error {
	// TODO: 发送轻量请求验证 API 可用性
	return nil
}

func (p *openaiProvider) EstimateCost(ctx context.Context, req *Request) (*Cost, error) {
	// TODO: 基于 token 数和定价计算成本
	return &Cost{}, nil
}
