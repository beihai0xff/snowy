package llm

import (
	"context"
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

func TestUnsupportedProviderNameDefaultsToUnconfigured(t *testing.T) {
	provider := NewUnsupportedProvider("  ")

	assert.Equal(t, "unconfigured", provider.Name())
	_, err := provider.Generate(context.Background(), &Request{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unconfigured provider: not implemented")
}
