// Package domain 定义物理建模域的领域模型。
// 有界上下文：Physics Modeling — 条件抽取、模型识别、推导、数值计算、图表。
// 参考技术方案 §15。
package domain

// ModelType 物理模型类型。
type ModelType string

const (
	ModelProjectileMotion    ModelType = "projectile_motion"
	ModelUniformMotion       ModelType = "uniform_motion"
	ModelUniformAcceleration ModelType = "uniform_acceleration"
	ModelNewtonSecondLaw     ModelType = "newton_second_law"
	ModelWorkEnergy          ModelType = "work_energy"
	ModelSpringOscillator    ModelType = "spring_oscillator"
	ModelTwoBodyMotion       ModelType = "two_body_motion"
)

// Condition 物理条件（已知量/未知量），参考技术方案 §15.3。
type Condition struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// DerivationStep 推导步骤。
type DerivationStep struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ParameterSchema 可调参数描述，参考技术方案 §15.5。
type ParameterSchema struct {
	Name    string  `json:"name"`
	Label   string  `json:"label"`
	Default float64 `json:"default"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Step    float64 `json:"step"`
	Unit    string  `json:"unit"`
}

// ChartSpec 2D 图表协议，参考技术方案 §15.4。
type ChartSpec struct {
	ChartType string       `json:"chart_type"` // line / scatter / bar
	Title     string       `json:"title"`
	XAxis     AxisSpec     `json:"x_axis"`
	YAxis     AxisSpec     `json:"y_axis"`
	Series    []SeriesSpec `json:"series"`
}

// AxisSpec 坐标轴描述。
type AxisSpec struct {
	Label string `json:"label"`
	Unit  string `json:"unit"`
}

// SeriesSpec 数据系列。
type SeriesSpec struct {
	Name string      `json:"name"`
	Data [][]float64 `json:"data"` // [[x, y], ...]
}

// PhysicsModel 物理建模完整结果，参考技术方案 §15.3。
type PhysicsModel struct {
	ModelType     ModelType         `json:"model_type"`
	Conditions    []Condition       `json:"conditions"`
	Steps         []DerivationStep  `json:"steps"`
	ResultSummary string            `json:"result_summary"`
	Warnings      []string          `json:"warnings,omitempty"`
	Chart         *ChartSpec        `json:"chart,omitempty"`
	Parameters    []ParameterSchema `json:"parameters,omitempty"`
}

// ComputeResult 数值计算结果。
type ComputeResult struct {
	Values   map[string]float64 `json:"values"`
	Chart    *ChartSpec         `json:"chart"`
	Warnings []string           `json:"warnings,omitempty"`
}
