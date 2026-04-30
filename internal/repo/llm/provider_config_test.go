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

func TestMiMoProviderRequiresConfiguredModel(t *testing.T) {
	provider := NewMiMoProvider(config.ModelProviderConfig{
		APIKey:        "test-key",
		BaseURL:       "https://example.test/v1",
		ModelProvider: "mimo",
		Timeout:       time.Second,
	})

	_, err := provider.Generate(context.Background(), &Request{Messages: []Message{{Role: "user", Content: "hello"}}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is empty")
}

func TestMiMoProviderRequiresConfiguredModelProvider(t *testing.T) {
	provider := NewMiMoProvider(config.ModelProviderConfig{
		APIKey:  "test-key",
		BaseURL: "https://example.test/v1",
		Model:   "mimo-test-model",
		Timeout: time.Second,
	})

	_, err := provider.Generate(context.Background(), &Request{Messages: []Message{{Role: "user", Content: "hello"}}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "model_provider is empty")
}

func TestMiMoProviderRequiresConfiguredBaseURL(t *testing.T) {
	provider := NewMiMoProvider(config.ModelProviderConfig{
		APIKey:        "test-key",
		Model:         "mimo-test-model",
		ModelProvider: "mimo",
		Timeout:       time.Second,
	})

	_, err := provider.Generate(context.Background(), &Request{Messages: []Message{{Role: "user", Content: "hello"}}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url is empty")
}

func TestMiMoProviderUsesRequestModelButStillRequiresConfiguredBaseURLAndModelProvider(t *testing.T) {
	provider := NewMiMoProvider(config.ModelProviderConfig{
		APIKey:        "test-key",
		ModelProvider: "mimo",
		Timeout:       time.Second,
	})

	_, err := provider.Generate(context.Background(), &Request{Model: "request-model", Messages: []Message{{Role: "user", Content: "hello"}}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url is empty")
	assert.NotContains(t, err.Error(), "model is empty")
}

func TestOpenAIProviderRequiresConfiguredBaseURL(t *testing.T) {
	provider := NewOpenAIProvider(config.ModelProviderConfig{
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: time.Second,
	})

	_, err := provider.Generate(context.Background(), &Request{Messages: []Message{{Role: "user", Content: "hello"}}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url is empty")
}

func TestConfiguredProviderMetadataIsTrimmed(t *testing.T) {
	provider := NewMiMoProvider(config.ModelProviderConfig{
		Model:         "  mimo-test-model  ",
		BaseURL:       " https://example.test/v1/ ",
		ModelProvider: "  mimo  ",
	})

	configured, ok := provider.(ConfiguredProvider)
	require.True(t, ok)
	assert.Equal(t, "mimo-test-model", configured.ConfiguredModel())
	assert.Equal(t, "https://example.test/v1", configured.ConfiguredBaseURL())
	assert.Equal(t, "mimo", configured.ConfiguredModelProvider())
	assert.False(t, strings.HasSuffix(configured.ConfiguredBaseURL(), "/"))
}

func TestUnsupportedProviderNameDefaultsToUnconfigured(t *testing.T) {
	provider := NewUnsupportedProvider("  ")

	assert.Equal(t, "unconfigured", provider.Name())
	_, err := provider.Generate(context.Background(), &Request{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unconfigured provider: not implemented")
}
