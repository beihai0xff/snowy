//nolint:cyclop // Provider request execution keeps validation, HTTP, and decoding branches together.
package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// openaiProvider 基于 OpenAI-compatible Chat Completions 协议调用模型网关。
// 厂商差异通过配置的 base_url、model 与可选 model_provider 表达；
// 密钥统一从本地运行时配置 llm.models[].api_key 注入。
type openaiProvider struct {
	unsupportedProvider

	cfg config.ModelProviderConfig
}

// NewOpenAIProvider 创建 OpenAI-compatible Provider。
func NewOpenAIProvider(cfg config.ModelProviderConfig) Provider {
	return &openaiProvider{
		unsupportedProvider: unsupportedProvider{name: "openai"},
		cfg:                 cfg,
	}
}

func (p *openaiProvider) ConfiguredModel() string {
	return p.cfg.EffectiveModel()
}

func (p *openaiProvider) ConfiguredBaseURL() string {
	return strings.TrimRight(p.cfg.EffectiveBaseURL(), "/")
}

func (p *openaiProvider) ConfiguredModelProvider() string {
	return strings.TrimSpace(p.cfg.ModelProvider)
}

func (p *openaiProvider) Generate(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		req = &Request{}
	}

	apiKey := p.apiKey()
	if apiKey == "" {
		return nil, errors.New(
			"openai-compatible provider: api key is empty; set llm.models[].api_key",
		)
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = p.cfg.EffectiveModel()
	}

	if model == "" {
		return nil, errors.New("openai-compatible provider: model is empty")
	}

	baseURL := p.ConfiguredBaseURL()
	if baseURL == "" {
		return nil, errors.New("openai-compatible provider: base_url is empty")
	}

	maxTokens := p.effectiveMaxTokens(req.MaxTokens)
	temperature := p.effectiveTemperature(req.Temperature)

	timeout := p.cfg.Timeout
	if reqCtxDeadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(reqCtxDeadline); remaining > 0 && (timeout <= 0 || remaining < timeout) {
			timeout = remaining
		}
	}

	if timeout <= 0 {
		timeout = 10 * time.Minute
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
		option.WithRequestTimeout(timeout),
	)

	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: toOpenAIChatMessages(req.Messages),
	}
	if maxTokens > 0 {
		params.MaxTokens = param.NewOpt(int64(maxTokens))
	}

	if temperature > 0 {
		params.Temperature = param.NewOpt(temperature)
	}

	if modelProvider := p.ConfiguredModelProvider(); modelProvider != "" {
		params.SetExtraFields(map[string]any{"model_provider": modelProvider})
	}

	resp, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, sdkProviderError(err)
	}

	if resp == nil || len(resp.Choices) == 0 {
		return nil, errors.New("openai-compatible provider: empty choices")
	}

	return &Response{
		Content:      resp.Choices[0].Message.Content,
		Model:        firstNonEmpty(resp.Model, model),
		InputTokens:  int(resp.Usage.PromptTokens),
		OutputTokens: int(resp.Usage.CompletionTokens),
		FinishReason: resp.Choices[0].FinishReason,
	}, nil
}

func toOpenAIChatMessages(messages []Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, message := range messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "system":
			out = append(out, openai.SystemMessage(content))
		case "assistant":
			out = append(out, openai.AssistantMessage(content))
		case "developer":
			out = append(out, openai.DeveloperMessage(content))
		default:
			out = append(out, openai.UserMessage(content))
		}
	}

	return out
}

func sdkProviderError(err error) error {
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		retryable := apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500

		return NewProviderError(
			fmt.Sprintf("openai-compatible provider: http status %d: %s", apiErr.StatusCode, apiErr.Message),
			retryable,
		)
	}

	return NewProviderError(fmt.Sprintf("openai-compatible provider: request failed: %v", err), true)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func (p *openaiProvider) apiKey() string {
	return strings.TrimSpace(p.cfg.APIKey)
}

func (p *openaiProvider) effectiveTemperature(requestTemperature float64) float64 {
	if p.cfg.Temperature == 0 {
		return 0
	}

	if requestTemperature > 0 {
		return requestTemperature
	}

	return p.cfg.Temperature
}

func (p *openaiProvider) effectiveMaxTokens(requestMaxTokens int) int {
	if p.cfg.MaxTokens > 0 {
		if requestMaxTokens > 0 && requestMaxTokens < p.cfg.MaxTokens {
			return requestMaxTokens
		}

		return p.cfg.MaxTokens
	}

	if requestMaxTokens > 0 {
		return requestMaxTokens
	}

	return MaxTokens128K
}
