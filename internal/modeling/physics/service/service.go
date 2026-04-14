// Package service 定义物理建模域的应用服务。
package service

import (
	"context"

	"github.com/beihai0xff/snowy/internal/modeling/physics/domain"
)

// PhysicsService 物理建模域应用服务接口。
type PhysicsService interface {
	// Analyze 分析物理题目，返回结构化建模结果。
	Analyze(ctx context.Context, question string, sessionContext string) (*domain.PhysicsModel, error)
	// Simulate 根据参数执行数值计算并返回更新后的图表。
	Simulate(ctx context.Context, modelType domain.ModelType, params map[string]float64) (*domain.ComputeResult, error)
}
