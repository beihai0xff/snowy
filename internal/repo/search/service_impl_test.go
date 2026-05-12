package search

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
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

type fakeAnswerRecordRepo struct {
	saveFn        func(ctx context.Context, record *AnswerRecord) error
	listByQueryFn func(ctx context.Context, query string, offset, limit int) ([]*AnswerRecord, int64, error)
}

func (f fakeAnswerRecordRepo) Save(ctx context.Context, record *AnswerRecord) error {
	if f.saveFn != nil {
		return f.saveFn(ctx, record)
	}
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	return nil
}

func (f fakeAnswerRecordRepo) GetByID(context.Context, string) (*AnswerRecord, error) {
	return nil, nil
}

func (f fakeAnswerRecordRepo) ListByUser(context.Context, string, int, int) ([]*AnswerRecord, int64, error) {
	return nil, 0, nil
}

func (f fakeAnswerRecordRepo) ListByQuery(ctx context.Context, query string, offset, limit int) ([]*AnswerRecord, int64, error) {
	if f.listByQueryFn != nil {
		return f.listByQueryFn(ctx, query, offset, limit)
	}
	return nil, 0, nil
}

type fakeFeedbackRepo struct {
	summary map[string]FeedbackSummary
}

func (f fakeFeedbackRepo) TargetFeedback(_ context.Context, targetType string, targetID string) (FeedbackSummary, error) {
	if f.summary == nil {
		return FeedbackSummary{}, nil
	}
	return f.summary[targetType+":"+targetID], nil
}

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

func TestService_QueryReturnsErrorWhenRepositoryFails(t *testing.T) {
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

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "search repository")
	assert.Contains(t, err.Error(), "opensearch unavailable")
}

func TestService_QueryUsesLLMDirectAnswer(t *testing.T) {
	repoCalled := false
	userID := uuid.New()
	var captured *llm.Request
	var saved *AnswerRecord
	svc := NewService(
		fakeSearchRepo{searchFn: func(context.Context, *ParsedQuery, Filters, int, int) ([]Result, int64, error) {
			repoCalled = true
			return nil, 0, nil
		}},
		fakeParser{},
		fakeRanker{},
		nil,
		nil,
		WithLLMProvider(fakeLLMProvider{name: "openai", model: "configured-gateway-model", generateFn: func(_ context.Context, req *llm.Request) (*llm.Response, error) {
			captured = req
			return &llm.Response{Content: "结论：牛顿第二定律说明物体加速度与合外力成正比。"}, nil
		}}),
		WithAnswerRecordRepository(fakeAnswerRecordRepo{saveFn: func(_ context.Context, record *AnswerRecord) error {
			saved = record
			record.ID = uuid.New()
			return nil
		}}),
	)

	resp, err := svc.Query(context.Background(), &Query{UserID: userID, Text: "牛顿第二定律是什么", Filters: Filters{Subject: "physics"}})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, repoCalled)
	assert.Contains(t, resp.Answer, "牛顿第二定律")
	require.NotEmpty(t, resp.Citations)
	assert.Equal(t, "runtime_grounding", resp.Citations[0].SourceType)
	assert.NotEmpty(t, resp.FormulaCards)
	assert.NotEmpty(t, resp.Misconceptions)
	assert.NotEmpty(t, resp.ExamMappings)
	assert.NotEmpty(t, resp.NextActions)
	assert.Contains(t, resp.KnowledgeTags, "大模型直答")
	assert.Contains(t, resp.KnowledgeTags, "openai")
	assert.GreaterOrEqual(t, resp.Confidence, 0.8)
	require.NotNil(t, captured)
	require.Len(t, captured.Messages, 2)
	assert.Equal(t, "system", captured.Messages[0].Role)
	assert.Contains(t, captured.Messages[0].Content, "不依赖外部检索结果")
	assert.NotContains(t, captured.Messages[0].Content, "Snowy")
	assert.NotContains(t, captured.Messages[0].Content, "学习平台")
	assert.True(t, strings.Contains(captured.Messages[1].Content, "不要进行数据库检索"))
	assert.Equal(t, "configured-gateway-model", captured.Model)
	require.NotNil(t, saved)
	assert.Equal(t, userID, saved.UserID)
	assert.Equal(t, "牛顿第二定律是什么", saved.Query)
	assert.Equal(t, "llm_direct", saved.Source)
	assert.Equal(t, "configured-gateway-model", saved.ModelName)
	assert.NotEmpty(t, saved.KnowledgeTags)
	assert.NotEmpty(t, saved.Citations)
	assert.NotEmpty(t, resp.AnswerID)
}

func TestService_QueryLLMFailureReturnsError(t *testing.T) {
	svc := NewService(
		nil,
		fakeParser{},
		fakeRanker{},
		nil,
		nil,
		WithLLMProvider(fakeLLMProvider{name: "openai", generateFn: func(context.Context, *llm.Request) (*llm.Response, error) {
			return nil, errors.New("model unavailable")
		}}),
	)

	resp, err := svc.Query(context.Background(), &Query{Text: "细胞膜有什么作用", Filters: Filters{Subject: "biology"}})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "llm direct answer")
	assert.Contains(t, err.Error(), "model unavailable")
}

func TestService_QueryUsesCommunityFeedbackForArchivedAnswer(t *testing.T) {
	goodID := uuid.New()
	badID := uuid.New()
	var saved *AnswerRecord
	svc := NewService(
		nil,
		fakeParser{},
		fakeRanker{},
		nil,
		nil,
		WithLLMProvider(fakeLLMProvider{name: "openai", model: "m1", generateFn: func(context.Context, *llm.Request) (*llm.Response, error) {
			return &llm.Response{Content: "新答案"}, nil
		}}),
		WithAnswerRecordRepository(fakeAnswerRecordRepo{
			listByQueryFn: func(context.Context, string, int, int) ([]*AnswerRecord, int64, error) {
				return []*AnswerRecord{
					{ID: badID, Query: "能量守恒", AnswerSummary: "差评历史答案", Confidence: 0.9},
					{ID: goodID, Query: "能量守恒", AnswerSummary: "好评历史答案", Confidence: 0.7},
				}, 2, nil
			},
			saveFn: func(_ context.Context, record *AnswerRecord) error {
				saved = record
				record.ID = uuid.New()
				return nil
			},
		}),
		WithFeedbackRepository(fakeFeedbackRepo{summary: map[string]FeedbackSummary{
			"answer:" + badID.String():  {LikeCount: 0, DislikeCount: 4},
			"answer:" + goodID.String(): {LikeCount: 5, DislikeCount: 0},
		}}),
	)

	resp, err := svc.Query(context.Background(), &Query{Text: "能量守恒", Filters: Filters{Subject: "physics"}})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, resp.KnowledgeTags, "社区反馈排序")
	require.NotEmpty(t, resp.RelatedQuestions)
	assert.Equal(t, "community-reviewed-answer", resp.RelatedQuestions[0].ID)
	require.NotNil(t, saved)
	assert.Equal(t, goodID.String(), saved.Metadata["ranked_by_answer_id"])
}
