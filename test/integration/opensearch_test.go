//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/content"
	"github.com/beihai0xff/snowy/internal/pkg/config"
	osrepo "github.com/beihai0xff/snowy/internal/repo/opensearch"
	internalsearch "github.com/beihai0xff/snowy/internal/repo/search"
)

func TestOpenSearchAdapterIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetOpenSearch(ctx))

	adapter := osrepo.NewOpenSearchAdapter(integrationOpenSearchConfig())
	docID := uuid.New()
	chunks := []*content.Chunk{
		{
			ID:         uuid.New(),
			DocumentID: docID,
			ChunkIndex: 0,
			Content:    "牛顿第二定律描述了力、质量和加速度之间的关系。",
			Tags:       []string{"subject:physics", "grade:high_school", "chapter:mechanics", "source:textbook"},
			ChunkType:  "paragraph",
			CreatedAt:  time.Now(),
		},
		{
			ID:         uuid.New(),
			DocumentID: docID,
			ChunkIndex: 1,
			Content:    "斜面问题中常把重力分解到平行和垂直斜面的方向。",
			Tags:       []string{"subject:physics", "grade:high_school", "chapter:mechanics", "source:lecture"},
			ChunkType:  "paragraph",
			CreatedAt:  time.Now(),
		},
	}

	require.NoError(t, adapter.Index(ctx, chunks))

	results, total, err := adapter.Search(ctx, &internalsearch.ParsedQuery{Original: "牛顿第二定律"}, internalsearch.Filters{Subject: "physics"}, 0, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))
	assert.NotEmpty(t, results)
	assert.Equal(t, "physics", results[0].Subject)
	assert.Contains(t, results[0].Snippet, "牛顿第二定律")

	result, err := adapter.GetByDocID(ctx, docID.String())
	require.NoError(t, err)
	assert.Equal(t, docID.String(), result.DocID)
	assert.Equal(t, "physics", result.Subject)

	require.NoError(t, adapter.Delete(ctx, docID.String()))

	results, total, err = adapter.Search(ctx, &internalsearch.ParsedQuery{Original: "牛顿第二定律"}, internalsearch.Filters{Subject: "physics"}, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, results)
}

func TestOpenSearchAdapter_FilterByChapterIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetOpenSearch(ctx))

	adapter := osrepo.NewOpenSearchAdapter(config.OpenSearchConfig{
		Addresses: integrationOpenSearchConfig().Addresses,
		Username:  integrationOpenSearchConfig().Username,
		Password:  integrationOpenSearchConfig().Password,
	})

	chunks := []*content.Chunk{
		{
			ID:         uuid.New(),
			DocumentID: uuid.New(),
			ChunkIndex: 0,
			Content:    "光合作用需要光照、二氧化碳和叶绿体。",
			Tags:       []string{"subject:biology", "chapter:photosynthesis", "source:textbook"},
			ChunkType:  "paragraph",
			CreatedAt:  time.Now(),
		},
		{
			ID:         uuid.New(),
			DocumentID: uuid.New(),
			ChunkIndex: 0,
			Content:    "牛顿第二定律适用于经典力学中的低速运动。",
			Tags:       []string{"subject:physics", "chapter:mechanics", "source:textbook"},
			ChunkType:  "paragraph",
			CreatedAt:  time.Now(),
		},
	}

	require.NoError(t, adapter.Index(ctx, chunks))

	results, total, err := adapter.Search(ctx, &internalsearch.ParsedQuery{Original: "需要"}, internalsearch.Filters{Subject: "biology", Chapter: "photosynthesis"}, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, "biology", results[0].Subject)
	assert.Equal(t, "photosynthesis", results[0].Chapter)
}

func TestOpenSearchAdapter_VectorRetrievalIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetOpenSearch(ctx))

	adapter := osrepo.NewOpenSearchAdapter(integrationOpenSearchConfig())
	physicsDocID := uuid.New()
	biologyDocID := uuid.New()
	chunks := []*content.Chunk{
		{
			ID:         uuid.New(),
			DocumentID: physicsDocID,
			ChunkIndex: 0,
			Content:    "牛顿第二定律说明合力决定加速度。",
			Embedding:  []float64{0.99, 0.01, 0, 0},
			Tags:       []string{"subject:physics", "chapter:mechanics", "source:textbook"},
			ChunkType:  "paragraph",
			CreatedAt:  time.Now(),
		},
		{
			ID:         uuid.New(),
			DocumentID: biologyDocID,
			ChunkIndex: 0,
			Content:    "光合作用受到光照和二氧化碳浓度影响。",
			Embedding:  []float64{0.02, 0.98, 0, 0},
			Tags:       []string{"subject:biology", "chapter:photosynthesis", "source:textbook"},
			ChunkType:  "paragraph",
			CreatedAt:  time.Now(),
		},
	}

	require.NoError(t, adapter.Index(ctx, chunks))

	results, total, err := adapter.Search(ctx, &internalsearch.ParsedQuery{Embedding: []float64{1, 0, 0, 0}}, internalsearch.Filters{}, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.NotEmpty(t, results)
	assert.Equal(t, physicsDocID.String(), results[0].DocID)
	assert.Equal(t, "physics", results[0].Subject)
}

func TestOpenSearchAdapter_DSLKeywordsAndEntitiesIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetOpenSearch(ctx))

	adapter := osrepo.NewOpenSearchAdapter(config.OpenSearchConfig{
		Addresses: integrationOpenSearchConfig().Addresses,
		Username:  integrationOpenSearchConfig().Username,
		Password:  integrationOpenSearchConfig().Password,
	})

	chunks := []*content.Chunk{
		{
			ID:         uuid.New(),
			DocumentID: uuid.New(),
			ChunkIndex: 0,
			Content:    "受力分析常用于斜面和连接体问题。",
			Tags:       []string{"subject:physics", "chapter:mechanics", "source:lecture", "force_analysis"},
			ChunkType:  "paragraph",
			CreatedAt:  time.Now(),
		},
		{
			ID:         uuid.New(),
			DocumentID: uuid.New(),
			ChunkIndex: 0,
			Content:    "细胞呼吸在线粒体中进行并释放能量。",
			Tags:       []string{"subject:biology", "chapter:respiration", "source:textbook"},
			ChunkType:  "paragraph",
			CreatedAt:  time.Now(),
		},
	}

	require.NoError(t, adapter.Index(ctx, chunks))

	results, total, err := adapter.Search(ctx, &internalsearch.ParsedQuery{
		Keywords: []string{"受力分析"},
		Entities: []string{"physics", "mechanics"},
	}, internalsearch.Filters{Source: "lecture"}, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)
	assert.Equal(t, "physics", results[0].Subject)
	assert.Equal(t, "mechanics", results[0].Chapter)
}

func TestOpenSearchAdapter_HybridQueryIntegration(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, resetOpenSearch(ctx))

	adapter := osrepo.NewOpenSearchAdapter(integrationOpenSearchConfig())
	bestDocID := uuid.New()
	otherDocID := uuid.New()
	chunks := []*content.Chunk{
		{
			ID:         uuid.New(),
			DocumentID: bestDocID,
			ChunkIndex: 0,
			Content:    "牛顿第二定律在斜面受力分析中非常关键。",
			Embedding:  []float64{0.97, 0.03, 0, 0},
			Tags:       []string{"subject:physics", "chapter:mechanics", "source:lecture"},
			ChunkType:  "paragraph",
			CreatedAt:  time.Now(),
		},
		{
			ID:         uuid.New(),
			DocumentID: otherDocID,
			ChunkIndex: 0,
			Content:    "斜面题也会涉及动能定理。",
			Embedding:  []float64{0.40, 0.60, 0, 0},
			Tags:       []string{"subject:physics", "chapter:mechanics", "source:lecture"},
			ChunkType:  "paragraph",
			CreatedAt:  time.Now(),
		},
	}

	require.NoError(t, adapter.Index(ctx, chunks))

	results, total, err := adapter.Search(ctx, &internalsearch.ParsedQuery{
		Original:  "牛顿第二定律 斜面",
		Embedding: []float64{1, 0, 0, 0},
	}, internalsearch.Filters{Subject: "physics", Source: "lecture"}, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.NotEmpty(t, results)
	assert.Equal(t, bestDocID.String(), results[0].DocID)
	assert.Contains(t, results[0].Snippet, "牛顿第二定律")
}
