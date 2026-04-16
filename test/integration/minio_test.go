//go:build integration

package integration

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/repo/storage"
)

func TestMinIOStorageIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetMinIO(ctx))

	store := storage.NewMinIOStorage(integrationMinIOConfig())
	key := "integration/" + uuid.NewString() + ".txt"
	content := "Snowy MinIO integration payload"

	require.NoError(t, store.Upload(ctx, key, strings.NewReader(content), "text/plain"))

	reader, err := store.Download(ctx, key)
	require.NoError(t, err)
	defer reader.Close()

	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, string(body))

	url, err := store.GetURL(ctx, key)
	require.NoError(t, err)
	assert.Contains(t, url, key)
	assert.Contains(t, url, "X-Amz-Algorithm")

	require.NoError(t, store.Delete(ctx, key))
	_, err = store.Download(ctx, key)
	assert.Error(t, err)
}

func TestMinIOStorageMultipleObjectsIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetMinIO(ctx))

	tests := []struct {
		name    string
		bucket  string
		key     string
		payload string
	}{
		{
			name:    "content",
			bucket:  "snowy-content",
			key:     "content/" + uuid.NewString() + ".md",
			payload: "content-payload",
		},
		{
			name:    "charts",
			bucket:  "snowy-charts",
			key:     "charts/" + uuid.NewString() + ".json",
			payload: `{"chart":true}`,
		},
		{
			name:    "exports",
			bucket:  "snowy-exports",
			key:     "exports/" + uuid.NewString() + ".txt",
			payload: "export-payload",
		},
	}

	for _, tt := range tests {
		store := storage.NewMinIOStorage(integrationMinIOBucketConfig(tt.bucket))
		require.NoError(
			t,
			store.Upload(ctx, tt.key, strings.NewReader(tt.payload), "application/octet-stream"),
			tt.name,
		)
		reader, err := store.Download(ctx, tt.key)
		require.NoError(t, err)
		body, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, tt.payload, string(body))
		_ = reader.Close()
	}

	require.NoError(t, resetMinIO(ctx))
	for _, tt := range tests {
		store := storage.NewMinIOStorage(integrationMinIOBucketConfig(tt.bucket))
		_, err := store.Download(ctx, tt.key)
		assert.Error(t, err)
	}
}
