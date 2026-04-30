package embedding

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

func TestOpenAIEmbeddingRequiresConfiguredBaseURLWhenAPIKeyPresent(t *testing.T) {
	provider := NewOpenAIEmbedding(config.EmbeddingConfig{
		APIKey:     "test-key",
		Model:      "embedding-test-model",
		Dimensions: 8,
	})

	_, err := provider.Embed(context.Background(), []string{"hello"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url is empty")
}

func TestOpenAIEmbeddingFallsBackToLocalWhenAPIKeyMissing(t *testing.T) {
	provider := NewOpenAIEmbedding(config.EmbeddingConfig{Dimensions: 8})

	vectors, err := provider.Embed(context.Background(), []string{"hello"})

	require.NoError(t, err)
	require.Len(t, vectors, 1)
	assert.Len(t, vectors[0], 8)
}
