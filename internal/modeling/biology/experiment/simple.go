package experiment

import (
	"context"
	"strings"

	"github.com/beihai0xff/snowy/internal/modeling/biology/domain"
)

type simpleAnalyzer struct{}

// NewSimpleAnalyzer 创建默认实验变量分析器。
func NewSimpleAnalyzer() Analyzer {
	return &simpleAnalyzer{}
}

func (a *simpleAnalyzer) AnalyzeVariables(_ context.Context, text string) (*domain.ExperimentVariables, error) {
	lower := strings.ToLower(text)
	vars := &domain.ExperimentVariables{}

	switch {
	case strings.Contains(lower, "光") || strings.Contains(lower, "photosynthesis"):
		vars.Independent = []string{"光照强度"}
		vars.Dependent = []string{"有机物积累"}
		vars.Controlled = []string{"温度", "二氧化碳浓度"}
	case strings.Contains(lower, "酶"):
		vars.Independent = []string{"温度"}
		vars.Dependent = []string{"酶活性"}
		vars.Controlled = []string{"pH", "底物浓度"}
	default:
		vars.Controlled = []string{"需补充实验条件"}
	}

	return vars, nil
}
