// Package embedding 定义 Embedding 供应商统一接口与适配实现（基础设施层）。
// 参考技术方案 §6.3.3。
package embedding

import "context"

// Provider Embedding 供应商统一接口。
type Provider interface {
	// Embed 将文本列表转为向量。
	Embed(ctx context.Context, texts []string) ([][]float64, error)
	// Dimensions 返回向量维度。
	Dimensions() int
}
