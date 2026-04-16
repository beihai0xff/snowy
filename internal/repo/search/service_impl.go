package search

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/beihai0xff/snowy/internal/repo/embedding"
)

const (
	defaultSearchOffset  = 0
	defaultSearchLimit   = 8
	maxRelatedTitleRunes = 48
)

type serviceImpl struct {
	repo      Repository
	parser    QueryParser
	ranker    ResultRanker
	embedding embedding.Provider
	logs      LogRepository
}

// NewService 创建知识检索服务实现。
func NewService(
	repo Repository,
	parser QueryParser,
	ranker ResultRanker,
	embeddingProvider embedding.Provider,
	logs LogRepository,
) Service {
	if logs == nil {
		logs = noopLogRepository{}
	}

	return &serviceImpl{
		repo:      repo,
		parser:    parser,
		ranker:    ranker,
		embedding: embeddingProvider,
		logs:      logs,
	}
}

func (s *serviceImpl) Query(ctx context.Context, q *Query) (*Response, error) {
	if q == nil || strings.TrimSpace(q.Text) == "" {
		return nil, errors.New("query text is empty")
	}

	if err := s.validateDependencies(); err != nil {
		return nil, err
	}

	start := time.Now()

	parsed, err := s.parser.Parse(q.Text)
	if err != nil {
		return nil, fmt.Errorf("parse query: %w", err)
	}

	s.attachEmbedding(ctx, q, parsed)

	results, total, err := s.repo.Search(ctx, parsed, q.Filters, defaultSearchOffset, defaultSearchLimit)
	if err != nil {
		return nil, fmt.Errorf("search repository: %w", err)
	}

	ranked := s.ranker.Rank(ctx, results, parsed)
	response := assembleResponse(ranked)

	if s.logs != nil {
		_ = s.logs.SaveLog(ctx, &Log{
			QueryText:   q.Text,
			ResultCount: int(total),
			LatencyMS:   int(time.Since(start).Milliseconds()),
			TopScore:    topScore(ranked),
		})
	}

	return response, nil
}

func assembleResponse(results []Result) *Response {
	if len(results) == 0 {
		return &Response{
			Answer:           "未找到直接匹配结果，建议补充更具体的学科、章节或关键词后再试。",
			KnowledgeTags:    nil,
			Citations:        nil,
			RelatedQuestions: nil,
			Confidence:       0.15,
		}
	}

	citations := make([]Citation, 0, minInt(len(results), 3))
	tagsSeen := map[string]struct{}{}
	tags := make([]string, 0, 6)
	related := make([]RelatedQuestion, 0, minInt(len(results), 3))

	snippets := make([]string, 0, minInt(len(results), 2))
	for i, result := range results {
		collectPrimaryResult(i, result, &citations, &related)
		collectSnippet(i, result, &snippets)
		tags = collectTags(result.Tags, tagsSeen, tags)
	}

	answer := strings.Join(snippets, "\n")
	if answer == "" {
		answer = "已检索到相关材料，但缺少足够片段用于生成摘要。"
	}

	return &Response{
		Answer:           answer,
		KnowledgeTags:    tags,
		Citations:        citations,
		RelatedQuestions: related,
		Confidence:       math.Min(0.99, math.Max(0.2, topScore(results))),
	}
}

func buildRelatedTitle(result Result) string {
	parts := make([]string, 0, 3)

	for _, part := range []string{result.Subject, result.Chapter, strings.TrimSpace(result.Snippet)} {
		if part == "" {
			continue
		}

		parts = append(parts, part)
		if len(parts) == 3 {
			break
		}
	}

	title := strings.Join(parts, " · ")
	if len([]rune(title)) > maxRelatedTitleRunes {
		title = string([]rune(title)[:maxRelatedTitleRunes]) + "…"
	}

	return title
}

func topScore(results []Result) float64 {
	if len(results) == 0 {
		return 0
	}

	if results[0].Score <= 0 {
		return 0.35
	}

	return math.Min(0.99, results[0].Score)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

type noopLogRepository struct{}

func (noopLogRepository) SaveLog(context.Context, *Log) error { return nil }

func (s *serviceImpl) validateDependencies() error {
	if s.repo == nil {
		return errors.New("search repository is nil")
	}

	if s.parser == nil {
		return errors.New("search parser is nil")
	}

	if s.ranker == nil {
		return errors.New("search ranker is nil")
	}

	return nil
}

func (s *serviceImpl) attachEmbedding(ctx context.Context, q *Query, parsed *ParsedQuery) {
	if s.embedding == nil {
		return
	}

	vectors, err := s.embedding.Embed(ctx, []string{q.Text})
	if err != nil || len(vectors) == 0 {
		return
	}

	parsed.Embedding = vectors[0]
}

func collectPrimaryResult(index int, result Result, citations *[]Citation, related *[]RelatedQuestion) {
	if index >= 3 {
		return
	}

	*citations = append(*citations, Citation{
		DocID:      result.DocID,
		SourceType: result.SourceType,
		Snippet:    result.Snippet,
		Score:      result.Score,
	})
	*related = append(*related, RelatedQuestion{
		ID:    result.DocID,
		Title: buildRelatedTitle(result),
	})
}

func collectSnippet(index int, result Result, snippets *[]string) {
	if index >= 2 || strings.TrimSpace(result.Snippet) == "" {
		return
	}

	*snippets = append(*snippets, strings.TrimSpace(result.Snippet))
}

func collectTags(resultTags []string, tagsSeen map[string]struct{}, tags []string) []string {
	for _, tag := range resultTags {
		if _, ok := tagsSeen[tag]; ok || strings.TrimSpace(tag) == "" {
			continue
		}

		tagsSeen[tag] = struct{}{}

		tags = append(tags, tag)
		if len(tags) == 6 {
			return tags
		}
	}

	return tags
}
