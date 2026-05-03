package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// mimoProvider 基于小米 MiMo OpenAI-compatible Chat Completions API。
type mimoProvider struct {
	unsupportedProvider

	cfg config.ModelProviderConfig
}

// NewMiMoProvider 创建小米 MiMo Provider。
func NewMiMoProvider(cfg config.ModelProviderConfig) Provider {
	return &mimoProvider{
		unsupportedProvider: unsupportedProvider{name: "mimo"},
		cfg:                 cfg,
	}
}

type miMoChatCompletionRequest struct {
	Model               string    `json:"model"`
	ModelProvider       string    `json:"model_provider,omitempty"`
	Messages            []Message `json:"messages"`
	MaxCompletionTokens int       `json:"max_completion_tokens,omitempty"`
	Temperature         float64   `json:"temperature,omitempty"`
	TopP                float64   `json:"top_p,omitempty"`
	Stream              bool      `json:"stream"`
	Stop                any       `json:"stop"`
	FrequencyPenalty    float64   `json:"frequency_penalty"`
	PresencePenalty     float64   `json:"presence_penalty"`
}

func (p *mimoProvider) ConfiguredModel() string {
	return p.cfg.EffectiveModel()
}

func (p *mimoProvider) ConfiguredBaseURL() string {
	return strings.TrimRight(p.cfg.EffectiveBaseURL(), "/")
}

func (p *mimoProvider) ConfiguredModelProvider() string {
	return strings.TrimSpace(p.cfg.ModelProvider)
}

type miMoChatCompletionResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens            int `json:"prompt_tokens"`
		CompletionTokens        int `json:"completion_tokens"`
		CompletionTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
	} `json:"usage"`
}

func (p *mimoProvider) Generate(ctx context.Context, req *Request) (*Response, error) {
	apiKey := firstNonEmpty(
		p.cfg.APIKey,
		os.Getenv("MIMO_API_KEY"),
		os.Getenv("XIAOMI_MIMO_API_KEY"),
		os.Getenv("SNOWY_LLM_PRIMARY_API_KEY"),
	)
	if apiKey == "" {
		return nil, fmt.Errorf("mimo provider: api key is empty; set MIMO_API_KEY or SNOWY_LLM_PRIMARY_API_KEY at runtime")
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = p.cfg.EffectiveModel()
	}
	if model == "" {
		return nil, fmt.Errorf("mimo provider: model is empty")
	}

	modelProvider := strings.TrimSpace(p.cfg.ModelProvider)
	if modelProvider == "" {
		return nil, fmt.Errorf("mimo provider: model_provider is empty")
	}

	baseURL := p.ConfiguredBaseURL()
	if baseURL == "" {
		return nil, fmt.Errorf("mimo provider: base_url is empty")
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = MaxTokens128K
	}

	temperature := req.Temperature
	if temperature <= 0 {
		temperature = 0.2
	}

	payload := miMoChatCompletionRequest{
		Model:               model,
		ModelProvider:       modelProvider,
		Messages:            req.Messages,
		MaxCompletionTokens: maxTokens,
		Temperature:         temperature,
		TopP:                0.95,
		Stream:              false,
		Stop:                nil,
		FrequencyPenalty:    0,
		PresencePenalty:     0,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	timeout := p.cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: timeout}).Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("mimo provider: http status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var decoded miMoChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if len(decoded.Choices) == 0 {
		return nil, fmt.Errorf("mimo provider: empty choices")
	}

	responseModel := decoded.Model
	if responseModel == "" {
		responseModel = model
	}

	choice := decoded.Choices[0]
	content := choice.Message.Content
	if strings.TrimSpace(content) == "" && choice.FinishReason == "length" {
		return nil, fmt.Errorf(
			"mimo provider: empty content after token budget exhausted (completion_tokens=%d, reasoning_tokens=%d)",
			decoded.Usage.CompletionTokens,
			decoded.Usage.CompletionTokensDetails.ReasoningTokens,
		)
	}

	return &Response{
		Content:      content,
		Model:        responseModel,
		InputTokens:  decoded.Usage.PromptTokens,
		OutputTokens: decoded.Usage.CompletionTokens,
		FinishReason: choice.FinishReason,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}
