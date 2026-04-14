// Package renderer 定义物理图表渲染接口。
package renderer

import "github.com/beihai0xff/snowy/internal/modeling/physics/domain"

// ChartRenderer 图表协议生成接口。
// 负责将计算结果转换为前端可渲染的 ChartSpec。
type ChartRenderer interface {
	// Render 根据计算结果生成图表协议。
	Render(result *domain.ComputeResult, modelType domain.ModelType) (*domain.ChartSpec, error)
}
