package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

func TestOpenAICompatibleProviderRequiresConfiguredModel(t *testing.T) {
	provider := NewOpenAIProvider(config.ModelProviderConfig{
		Provider:      "openai",
		APIKey:        "test-key",
		BaseURL:       "https://example.test/v1",
		ModelProvider: "gateway",
		Timeout:       time.Second,
	})

	_, err := provider.Generate(context.Background(), &Request{Messages: []Message{{Role: "user", Content: "hello"}}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is empty")
}

func TestOpenAICompatibleProviderRequiresConfiguredBaseURL(t *testing.T) {
	provider := NewOpenAIProvider(config.ModelProviderConfig{
		Provider:      "openai",
		APIKey:        "test-key",
		Model:         "gateway-test-model",
		ModelProvider: "gateway",
		Timeout:       time.Second,
	})

	_, err := provider.Generate(context.Background(), &Request{Messages: []Message{{Role: "user", Content: "hello"}}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url is empty")
}

func TestOpenAICompatibleProviderUsesRequestModel(t *testing.T) {
	provider := NewOpenAIProvider(config.ModelProviderConfig{
		Provider:      "openai",
		APIKey:        "test-key",
		ModelProvider: "gateway",
		Timeout:       time.Second,
	})

	_, err := provider.Generate(context.Background(), &Request{Model: "request-model", Messages: []Message{{Role: "user", Content: "hello"}}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url is empty")
	assert.NotContains(t, err.Error(), "model is empty")
}

func TestOpenAICompatibleProviderMetadataIsTrimmed(t *testing.T) {
	provider := NewOpenAIProvider(config.ModelProviderConfig{
		Provider:      "  openai  ",
		Model:         "  gateway-test-model  ",
		BaseURL:       " https://example.test/v1/ ",
		ModelProvider: "  gateway  ",
	})

	configured, ok := provider.(ConfiguredProvider)
	require.True(t, ok)
	assert.Equal(t, "gateway-test-model", configured.ConfiguredModel())
	assert.Equal(t, "https://example.test/v1", configured.ConfiguredBaseURL())
	assert.Equal(t, "gateway", configured.ConfiguredModelProvider())
	assert.False(t, strings.HasSuffix(configured.ConfiguredBaseURL(), "/"))
	assert.Equal(t, "openai", provider.Name())
}

func TestOpenAICompatibleProviderCanOmitTemperatureByConfig(t *testing.T) {
	provider := NewOpenAIProvider(config.ModelProviderConfig{
		Provider:    "openai",
		Model:       "gpt-5.5-2026-04-24",
		BaseURL:     "https://example.test/v1",
		APIKey:      "test-key",
		Temperature: 0,
	})

	openaiProvider, ok := provider.(*openaiProvider)
	require.True(t, ok)

	assert.Zero(t, openaiProvider.effectiveTemperature(0.2))
}

func TestOpenAICompatibleProviderUsesRequestTemperatureWhenConfigAllows(t *testing.T) {
	provider := NewOpenAIProvider(config.ModelProviderConfig{
		Provider:    "openai",
		Model:       "gateway-test-model",
		BaseURL:     "https://example.test/v1",
		APIKey:      "test-key",
		Temperature: 0.7,
	})

	openaiProvider, ok := provider.(*openaiProvider)
	require.True(t, ok)

	assert.Equal(t, 0.2, openaiProvider.effectiveTemperature(0.2))
	assert.Equal(t, 0.7, openaiProvider.effectiveTemperature(0))
}

func TestOpenAICompatibleProviderCapsMaxTokensByConfig(t *testing.T) {
	provider := NewOpenAIProvider(config.ModelProviderConfig{
		Provider:  "openai",
		Model:     "gpt-5.5-2026-04-24",
		BaseURL:   "https://example.test/v1",
		APIKey:    "test-key",
		MaxTokens: 128000,
	})

	openaiProvider, ok := provider.(*openaiProvider)
	require.True(t, ok)

	assert.Equal(t, 128000, openaiProvider.effectiveMaxTokens(MaxTokens128K))
	assert.Equal(t, 8192, openaiProvider.effectiveMaxTokens(8192))
	assert.Equal(t, 128000, openaiProvider.effectiveMaxTokens(0))
}

func TestOpenAICompatibleProviderUsesRequestMaxTokensWithoutConfigCap(t *testing.T) {
	provider := NewOpenAIProvider(config.ModelProviderConfig{
		Provider: "openai",
		Model:    "gateway-test-model",
		BaseURL:  "https://example.test/v1",
		APIKey:   "test-key",
	})

	openaiProvider, ok := provider.(*openaiProvider)
	require.True(t, ok)

	assert.Equal(t, 2048, openaiProvider.effectiveMaxTokens(2048))
	assert.Equal(t, MaxTokens128K, openaiProvider.effectiveMaxTokens(0))
}

func TestOpenAICompatibleProviderUsesSDKRequestShape(t *testing.T) {
	var capturedAuth string
	var capturedPath string
	var captured map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedPath = r.URL.Path

		require.NoError(t, json.NewDecoder(r.Body).Decode(&captured))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-test",
			"object":"chat.completion",
			"created":1,
			"model":"sdk-model",
			"choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}],
			"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}
		}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider(config.ModelProviderConfig{
		Provider:      "openai",
		Model:         "gpt-5.5-2026-04-24",
		ModelProvider: "bytedance",
		BaseURL:       server.URL,
		APIKey:        "test-key",
		Temperature:   0,
		MaxTokens:     128000,
		Timeout:       time.Second,
	})

	resp, err := provider.Generate(context.Background(), &Request{
		MaxTokens:   MaxTokens128K,
		Temperature: 0.2,
		Messages: []Message{
			{Role: "system", Content: "system prompt"},
			{Role: "user", Content: "hello"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Content)
	assert.Equal(t, "sdk-model", resp.Model)
	assert.Equal(t, 3, resp.InputTokens)
	assert.Equal(t, 2, resp.OutputTokens)
	assert.Equal(t, "stop", resp.FinishReason)

	assert.Equal(t, "Bearer test-key", capturedAuth)
	assert.Equal(t, "/chat/completions", capturedPath)
	assert.Equal(t, "gpt-5.5-2026-04-24", captured["model"])
	assert.Equal(t, "bytedance", captured["model_provider"])
	assert.Equal(t, float64(128000), captured["max_tokens"])
	assert.NotContains(t, captured, "temperature")
}

func TestUnsupportedProviderNameDefaultsToUnconfigured(t *testing.T) {
	provider := NewUnsupportedProvider("  ")

	assert.Equal(t, "unconfigured", provider.Name())
	_, err := provider.Generate(context.Background(), &Request{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unconfigured provider: not implemented")
}
