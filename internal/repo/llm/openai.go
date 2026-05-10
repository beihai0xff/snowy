package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

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

type openAIChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
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

type openAIChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (p *openaiProvider) Generate(ctx context.Context, req *Request) (*Response, error) {
	apiKey := firstNonEmpty(
		p.cfg.APIKey,
		os.Getenv("OPENAI_API_KEY"),
		os.Getenv("SNOWY_LLM_PRIMARY_API_KEY"),
		os.Getenv("SNOWY_LLM_FALLBACK_API_KEY"),
	)
	if apiKey == "" {
		return nil, errors.New("openai provider: api key is empty; set api_key, OPENAI_API_KEY, or SNOWY_LLM_*_API_KEY")
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = p.cfg.EffectiveModel()
	}

	if model == "" {
		return nil, errors.New("openai provider: model is empty")
	}

	baseURL := p.ConfiguredBaseURL()
	if baseURL == "" {
		return nil, errors.New("openai provider: base_url is empty")
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = p.cfg.MaxTokens
	}
	if maxTokens <= 0 {
		maxTokens = MaxTokens128K
	}

	temperature := req.Temperature
	if temperature <= 0 {
		temperature = p.cfg.Temperature
	}

	payload := openAIChatCompletionRequest{
		Model:       model,
		Messages:    req.Messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	timeout := p.cfg.Timeout
	if reqCtxDeadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(reqCtxDeadline); remaining > 0 && (timeout <= 0 || remaining < timeout) {
			timeout = remaining
		}
	}

	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: timeout}).Do(httpReq)
	if err != nil {
		return nil, NewProviderError(fmt.Sprintf("openai provider: request failed: %v", err), true)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError

		return nil, NewProviderError(fmt.Sprintf("openai provider: http status %d", resp.StatusCode), retryable)
	}

	var decoded openAIChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	if len(decoded.Choices) == 0 {
		return nil, errors.New("openai provider: empty choices")
	}

	return &Response{
		Content:      decoded.Choices[0].Message.Content,
		Model:        model,
		InputTokens:  decoded.Usage.PromptTokens,
		OutputTokens: decoded.Usage.CompletionTokens,
		FinishReason: decoded.Choices[0].FinishReason,
	}, nil
}
