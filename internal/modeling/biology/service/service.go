// Package service 定义生物建模域的应用服务。
package service

import (
	"context"

	"github.com/beihai0xff/snowy/internal/modeling/biology/domain"
)

// BiologyService 生物建模域应用服务接口。
type BiologyService interface {
	// Analyze 分析生物问题，返回结构化建模结果。
	Analyze(ctx context.Context, question string, sessionContext string) (*domain.BiologyModel, error)
}
