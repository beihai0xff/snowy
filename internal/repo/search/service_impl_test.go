package search

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/repo/llm"
)

type fakeSearchRepo struct {
	searchFn func(ctx context.Context, query *ParsedQuery, filters Filters, offset, limit int) ([]Result, int64, error)
}

func (f fakeSearchRepo) Search(ctx context.Context, query *ParsedQuery, filters Filters, offset, limit int) ([]Result, int64, error) {
	return f.searchFn(ctx, query, filters, offset, limit)
}

func (f fakeSearchRepo) GetByDocID(context.Context, string) (*Result, error) { return nil, nil }

type fakeParser struct{}

func (fakeParser) Parse(raw string) (*ParsedQuery, error) {
	return &ParsedQuery{Original: raw, Keywords: []string{raw}, Intent: "explain"}, nil
}

type fakeLLMProvider struct {
	name       string
	model      string
	generateFn func(ctx context.Context, req *llm.Request) (*llm.Response, error)
}

func (f fakeLLMProvider) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	return f.generateFn(ctx, req)
}

func (f fakeLLMProvider) GenerateStream(context.Context, *llm.Request, chan<- llm.StreamChunk) error {
	return nil
}

func (f fakeLLMProvider) HealthCheck(context.Context) error { return nil }

func (f fakeLLMProvider) EstimateCost(context.Context, *llm.Request) (*llm.Cost, error) {
	return &llm.Cost{}, nil
}

func (f fakeLLMProvider) ConfiguredModel() string { return f.model }

func (f fakeLLMProvider) ConfiguredBaseURL() string { return "" }

func (f fakeLLMProvider) ConfiguredModelProvider() string { return "" }

func (f fakeLLMProvider) Name() string {
	if f.name == "" {
		return "fake-llm"
	}

	return f.name
}

type fakeRanker struct{}

func (fakeRanker) Rank(_ context.Context, results []Result, _ *ParsedQuery) []Result { return results }

func TestService_QueryFallbackWhenRepositoryFails(t *testing.T) {
	svc := NewService(
		fakeSearchRepo{searchFn: func(context.Context, *ParsedQuery, Filters, int, int) ([]Result, int64, error) {
			return nil, 0, errors.New("opensearch unavailable")
		}},
		fakeParser{},
		fakeRanker{},
		nil,
		nil,
	)

	resp, err := svc.Query(context.Background(), &Query{Text: "牛顿第二定律", Filters: Filters{Subject: "physics"}})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 0.15, resp.Confidence)
	assert.Contains(t, resp.Answer, "当前知识索引暂时不可用")
	assert.Contains(t, resp.Answer, "opensearch unavailable")
	assert.Contains(t, resp.KnowledgeTags, "本地兜底")
	assert.Contains(t, resp.KnowledgeTags, "physics")
	require.Len(t, resp.Citations, 1)
	assert.Equal(t, "local-fallback", resp.Citations[0].DocID)
	require.NotEmpty(t, resp.RelatedQuestions)
}

func TestService_QueryUsesLLMDirectAnswer(t *testing.T) {
	repoCalled := false
	var captured *llm.Request
	svc := NewService(
		fakeSearchRepo{searchFn: func(context.Context, *ParsedQuery, Filters, int, int) ([]Result, int64, error) {
			repoCalled = true
			return nil, 0, nil
		}},
		fakeParser{},
		fakeRanker{},
		nil,
		nil,
		WithLLMProviders(fakeLLMProvider{name: "mimo", model: "configured-mimo-model", generateFn: func(_ context.Context, req *llm.Request) (*llm.Response, error) {
			captured = req
			return &llm.Response{Content: "结论：牛顿第二定律说明物体加速度与合外力成正比。"}, nil
		}}, nil),
	)

	resp, err := svc.Query(context.Background(), &Query{Text: "牛顿第二定律是什么", Filters: Filters{Subject: "physics"}})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, repoCalled)
	assert.Contains(t, resp.Answer, "牛顿第二定律")
	assert.Empty(t, resp.Citations)
	assert.Contains(t, resp.KnowledgeTags, "大模型直答")
	assert.Contains(t, resp.KnowledgeTags, "mimo")
	assert.GreaterOrEqual(t, resp.Confidence, 0.8)
	require.NotNil(t, captured)
	require.Len(t, captured.Messages, 2)
	assert.Equal(t, "system", captured.Messages[0].Role)
	assert.Contains(t, captured.Messages[0].Content, "不使用 RAG 检索结果")
	assert.True(t, strings.Contains(captured.Messages[1].Content, "不要进行数据库检索"))
	assert.Equal(t, "configured-mimo-model", captured.Model)
}

func TestService_QueryLLMFailureReturnsFallback(t *testing.T) {
	svc := NewService(
		nil,
		fakeParser{},
		fakeRanker{},
		nil,
		nil,
		WithLLMProviders(fakeLLMProvider{name: "mimo", generateFn: func(context.Context, *llm.Request) (*llm.Response, error) {
			return nil, errors.New("model unavailable")
		}}, nil),
	)

	resp, err := svc.Query(context.Background(), &Query{Text: "细胞膜有什么作用", Filters: Filters{Subject: "biology"}})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 0.15, resp.Confidence)
	assert.Contains(t, resp.Answer, "当前大模型知识问答暂时不可用")
	assert.Contains(t, resp.Answer, "model unavailable")
	assert.Contains(t, resp.KnowledgeTags, "本地兜底")
	assert.Contains(t, resp.KnowledgeTags, "biology")
}
