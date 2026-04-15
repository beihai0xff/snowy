package ranking

import (
	"context"
	"sort"
	"strings"

	internalsearch "github.com/beihai0xff/snowy/internal/repo/search"
)

// scoreRanker 基于分数和关键字命中做稳定重排。
type scoreRanker struct{}

// NewScoreRanker 创建默认重排器。
func NewScoreRanker() Ranker {
	return &scoreRanker{}
}

func (r *scoreRanker) Rank(_ context.Context, results []internalsearch.Result, query *internalsearch.ParsedQuery) []internalsearch.Result {
	ranked := append([]internalsearch.Result(nil), results...)
	keywords := map[string]struct{}{}
	if query != nil {
		for _, keyword := range query.Keywords {
			keywords[strings.ToLower(keyword)] = struct{}{}
		}
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		left := ranked[i]
		right := ranked[j]
		leftBoost := keywordBoost(left, keywords)
		rightBoost := keywordBoost(right, keywords)
		if left.Score+leftBoost == right.Score+rightBoost {
			return left.DocID < right.DocID
		}
		return left.Score+leftBoost > right.Score+rightBoost
	})

	return ranked
}

func keywordBoost(result internalsearch.Result, keywords map[string]struct{}) float64 {
	if len(keywords) == 0 {
		return 0
	}

	corpus := strings.ToLower(strings.Join([]string{
		result.Subject,
		result.Chapter,
		result.Snippet,
		strings.Join(result.Tags, " "),
	}, " "))

	boost := 0.0
	for keyword := range keywords {
		if strings.Contains(corpus, keyword) {
			boost += 0.05
		}
	}

	return boost
}
