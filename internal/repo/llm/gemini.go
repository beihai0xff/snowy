package llm

import (
	"context"
	"fmt"

	"github.com/beihai0xff/snowy/internal/config"
)

// geminiProvider 基于 Google Gemini API 的 LLM Provider。
type geminiProvider struct {
	cfg config.ModelProviderConfig
}

// NewGeminiProvider 创建 Gemini Provider。
func NewGeminiProvider(cfg config.ModelProviderConfig) Provider {
	return &geminiProvider{cfg: cfg}
}

func (p *geminiProvider) Name() string { return "gemini" }

func (p *geminiProvider) Generate(ctx context.Context, req *Request) (*Response, error) {
	// TODO: 通过 Eino Gemini ChatModel 实现
	return nil, fmt.Errorf("gemini provider: not implemented")
}

func (p *geminiProvider) GenerateStream(ctx context.Context, req *Request, chunks chan<- StreamChunk) error {
	// TODO: 通过 Eino Gemini ChatModel streaming 实现
	return fmt.Errorf("gemini provider stream: not implemented")
}

func (p *geminiProvider) HealthCheck(ctx context.Context) error {
	return nil
}

func (p *geminiProvider) EstimateCost(ctx context.Context, req *Request) (*Cost, error) {
	return &Cost{}, nil
}
