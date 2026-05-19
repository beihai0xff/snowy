package search

import (
	"context"
	"sort"
	"strings"
)

// NewScoreRanker creates the default score-and-keyword based ranker.
func NewScoreRanker() ResultRanker {
	return &scoreRanker{}
}

type scoreRanker struct{}

func (r *scoreRanker) Rank(_ context.Context, results []Result, query *ParsedQuery) []Result {
	ranked := append([]Result(nil), results...)
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

func keywordBoost(result Result, keywords map[string]struct{}) float64 {
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
