// Package experiment 定义实验变量分析接口。
package experiment

import (
	"context"

	"github.com/beihai0xff/snowy/internal/modeling/biology/domain"
)

// Analyzer 实验变量分析接口。
// 参考技术方案 §16.5。
type Analyzer interface {
	// AnalyzeVariables 从实验题目文本中识别自变量、因变量、控制变量。
	AnalyzeVariables(ctx context.Context, text string) (*domain.ExperimentVariables, error)
}
