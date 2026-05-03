// Package llm 定义 LLM 供应商统一接口与适配实现（基础设施层）。
// 参考技术方案 §14.3 — 所有模型调用通过统一 Provider Adapter。
package llm

import (
	"context"
	"fmt"
	"strings"
)

// MaxTokens128K is the unified 128k generation token budget used for all model calls.
const MaxTokens128K = 128 * 1024

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

// ConfiguredProvider 暴露 Provider 从配置加载的模型元信息。
//
// 该接口是可选接口，避免测试桩和第三方 Provider 必须实现；调用方可在需要把
// configured model 显式传入 Request 时按需断言。模型名称、base_url、
// model_provider 等运行参数必须来自配置，而不是 Provider 内部硬编码默认值。
type ConfiguredProvider interface {
	ConfiguredModel() string
	ConfiguredBaseURL() string
	ConfiguredModelProvider() string
}

// NewUnsupportedProvider 创建一个显式不可用的 Provider，用于配置缺失或未知 provider 时
// 保持调用链可降级，而不是静默切到某个硬编码默认厂商。
func NewUnsupportedProvider(name string) Provider {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "unconfigured"
	}

	return unsupportedProvider{name: name}
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
