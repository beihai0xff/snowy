package embedding

import (
	"context"
	"errors"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// openaiEmbedding 基于 OpenAI text-embedding-3-large 的实现。
type openaiEmbedding struct {
	cfg config.EmbeddingConfig
}

// NewOpenAIEmbedding 创建 OpenAI Embedding Provider。
func NewOpenAIEmbedding(cfg config.EmbeddingConfig) Provider {
	return &openaiEmbedding{cfg: cfg}
}

func (e *openaiEmbedding) Embed(_ context.Context, _ []string) ([][]float64, error) {
	// TODO: 通过 Eino Embedding Provider 实现
	return nil, errors.New("openai embedding: not implemented")
}

func (e *openaiEmbedding) Dimensions() int {
	return e.cfg.Dimensions
}
