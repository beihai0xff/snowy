package search

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/beihai0xff/snowy/internal/repo/embedding"
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
		return nil, fmt.Errorf("query text is empty")
	}
	if s.repo == nil {
		return nil, fmt.Errorf("search repository is nil")
	}
	if s.parser == nil {
		return nil, fmt.Errorf("search parser is nil")
	}
	if s.ranker == nil {
		return nil, fmt.Errorf("search ranker is nil")
	}

	start := time.Now()
	parsed, err := s.parser.Parse(q.Text)
	if err != nil {
		return nil, fmt.Errorf("parse query: %w", err)
	}
	if s.embedding != nil {
		if vectors, embedErr := s.embedding.Embed(ctx, []string{q.Text}); embedErr == nil && len(vectors) > 0 {
			parsed.Embedding = vectors[0]
		}
	}

	results, total, err := s.repo.Search(ctx, parsed, q.Filters, 0, 8)
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

	citations := make([]Citation, 0, min(len(results), 3))
	tagsSeen := map[string]struct{}{}
	tags := make([]string, 0, 6)
	related := make([]RelatedQuestion, 0, min(len(results), 3))
	snippets := make([]string, 0, min(len(results), 2))
	for i, result := range results {
		if i < 3 {
			citations = append(citations, Citation{
				DocID:      result.DocID,
				SourceType: result.SourceType,
				Snippet:    result.Snippet,
				Score:      result.Score,
			})
			related = append(related, RelatedQuestion{
				ID:    result.DocID,
				Title: buildRelatedTitle(result),
			})
		}
		if i < 2 && strings.TrimSpace(result.Snippet) != "" {
			snippets = append(snippets, strings.TrimSpace(result.Snippet))
		}
		for _, tag := range result.Tags {
			if _, ok := tagsSeen[tag]; ok || strings.TrimSpace(tag) == "" {
				continue
			}
			tagsSeen[tag] = struct{}{}
			tags = append(tags, tag)
			if len(tags) == 6 {
				break
			}
		}
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
	if len([]rune(title)) > 48 {
		title = string([]rune(title)[:48]) + "…"
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
