// Package embedding 定义 Embedding 供应商适配层。
// 参考技术方案 §6.3.3。
package embedding

import (
	"context"
	"fmt"

	"github.com/beihai0xff/snowy/internal/config"
)

// Provider Embedding 供应商统一接口。
type Provider interface {
	// Embed 将文本列表转为向量。
	Embed(ctx context.Context, texts []string) ([][]float64, error)
	// Dimensions 返回向量维度。
	Dimensions() int
}

// openaiEmbedding 基于 OpenAI text-embedding-3-large 的实现。
type openaiEmbedding struct {
	cfg config.EmbeddingConfig
}

// NewOpenAIEmbedding 创建 OpenAI Embedding Provider。
func NewOpenAIEmbedding(cfg config.EmbeddingConfig) Provider {
	return &openaiEmbedding{cfg: cfg}
}

func (e *openaiEmbedding) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	// TODO: 通过 Eino Embedding Provider 实现
	return nil, fmt.Errorf("openai embedding: not implemented")
}

func (e *openaiEmbedding) Dimensions() int {
	return e.cfg.Dimensions
}
