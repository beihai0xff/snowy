// Package llm 定义 LLM 供应商统一接口与适配实现（基础设施层）。
// 参考技术方案 §14.3 — 所有模型调用通过统一 Provider Adapter。
package llm

import (
	"context"
	"fmt"
)

// Request LLM 调用请求。
type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// Message 对话消息。
type Message struct {
	Role    string `json:"role"` // system / user / assistant
	Content string `json:"content"`
}

// Response LLM 调用响应。
type Response struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	FinishReason string `json:"finish_reason"`
}

// Cost 成本预估。
type Cost struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
}

// StreamChunk 流式输出 chunk。
type StreamChunk struct {
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// Provider LLM 供应商统一接口。
// 屏蔽厂商 SDK 差异，统一超时、重试、限流、日志、审计。
type Provider interface {
	// Generate 同步调用 LLM。
	Generate(ctx context.Context, req *Request) (*Response, error)
	// GenerateStream 流式调用 LLM。
	GenerateStream(ctx context.Context, req *Request, chunks chan<- StreamChunk) error
	// HealthCheck 健康检查。
	HealthCheck(ctx context.Context) error
	// EstimateCost 预估成本。
	EstimateCost(ctx context.Context, req *Request) (*Cost, error)
	// Name 返回供应商名称。
	Name() string
}

type unsupportedProvider struct {
	name string
}

func (p unsupportedProvider) Name() string {
	return p.name
}

func (p unsupportedProvider) Generate(_ context.Context, _ *Request) (*Response, error) {
	return nil, fmt.Errorf("%s provider: not implemented", p.name)
}

func (p unsupportedProvider) GenerateStream(_ context.Context, _ *Request, _ chan<- StreamChunk) error {
	return fmt.Errorf("%s provider stream: not implemented", p.name)
}

func (p unsupportedProvider) HealthCheck(_ context.Context) error {
	return nil
}

func (p unsupportedProvider) EstimateCost(_ context.Context, _ *Request) (*Cost, error) {
	return &Cost{}, nil
}
