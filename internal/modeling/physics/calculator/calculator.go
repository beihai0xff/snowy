// Package calculator 定义物理数值计算接口。
package calculator

import "github.com/beihai0xff/snowy/internal/modeling/physics/domain"

// Calculator 物理数值计算引擎接口。
// 参考技术方案 §15.5：参数调节优先走轻量计算接口，不重复调用 LLM。
type Calculator interface {
	// Compute 根据物理模型和参数计算数值结果及图表数据。
	Compute(model domain.ModelType, params map[string]float64) (*domain.ComputeResult, error)
	// SupportedModels 返回支持的物理模型类型。
	SupportedModels() []domain.ModelType
}
