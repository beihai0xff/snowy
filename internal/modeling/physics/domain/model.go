// Package domain 定义物理建模域的领域模型。
// 有界上下文：Physics Modeling — 条件抽取、模型识别、推导、前端代码生成、浏览器渲染。
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

// RenderMode 浏览器渲染模式。
type RenderMode string

const (
	RenderModeHTMLIframe  RenderMode = "html_iframe"
	RenderModeReactIframe RenderMode = "react_iframe"
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

// ChartSpec 兼容旧版 2D 图表协议。
type ChartSpec struct {
	ChartType string       `json:"chart_type"`
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
	Data [][]float64 `json:"data"`
}

// SceneSpec 前端代码生成所需的场景规格。
type SceneSpec struct {
	SceneType    string             `json:"scene_type"`
	Title        string             `json:"title"`
	Summary      string             `json:"summary,omitempty"`
	RenderMode   RenderMode         `json:"render_mode,omitempty"`
	DefaultProps map[string]float64 `json:"default_props,omitempty"`
}

// RenderManifest 浏览器渲染清单。
type RenderManifest struct {
	Entry         string             `json:"entry"`
	Framework     string             `json:"framework"`
	Sandbox       string             `json:"sandbox"`
	RenderMode    RenderMode         `json:"render_mode"`
	MountSelector string             `json:"mount_selector"`
	Dependencies  []string           `json:"dependencies,omitempty"`
	AllowedAPIs   []string           `json:"allowed_apis,omitempty"`
	BlockedAPIs   []string           `json:"blocked_apis,omitempty"`
	InitialProps  map[string]float64 `json:"initial_props,omitempty"`
}

// RenderArtifact 可供前端直接渲染的代码产物。
type RenderArtifact struct {
	SceneType      string            `json:"scene_type"`
	RenderMode     RenderMode        `json:"render_mode"`
	RenderManifest *RenderManifest   `json:"render_manifest"`
	CodeBundle     map[string]string `json:"code_bundle"`
	ResultSummary  string            `json:"result_summary"`
	Warnings       []string          `json:"warnings,omitempty"`
}

// PhysicsModel 物理建模完整结果，兼容旧版图表协议，同时补充 scene_spec 供代码生成使用。
type PhysicsModel struct {
	ModelType     ModelType         `json:"model_type"`
	Conditions    []Condition       `json:"conditions"`
	Steps         []DerivationStep  `json:"steps"`
	ResultSummary string            `json:"result_summary"`
	Warnings      []string          `json:"warnings,omitempty"`
	Chart         *ChartSpec        `json:"chart,omitempty"`
	Parameters    []ParameterSchema `json:"parameters,omitempty"`
	SceneSpec     *SceneSpec        `json:"scene_spec,omitempty"`
}

// ComputeResult 数值计算结果（保留旧 simulate 能力，兼容历史链路）。
type ComputeResult struct {
	Values   map[string]float64 `json:"values"`
	Chart    *ChartSpec         `json:"chart"`
	Warnings []string           `json:"warnings,omitempty"`
}
