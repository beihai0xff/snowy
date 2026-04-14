// Package domain 定义生物建模域的领域模型。
// 有界上下文：Biology Modeling — 概念识别、关系抽取、过程拆解、实验变量分析。
// 参考技术方案 §16。
package domain

// Concept 生物概念实体，参考技术方案 §16.3。
type Concept struct {
	Name string `json:"name"`
	Type string `json:"type"` // factor / result / process / structure / substance
}

// Relation 概念关系，参考技术方案 §16.3。
type Relation struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // influences / produces / inhibits / composes / transforms
}

// ProcessStep 过程阶段，参考技术方案 §16.3。
type ProcessStep struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ExperimentVariables 实验变量分析，参考技术方案 §16.5。
type ExperimentVariables struct {
	Independent []string `json:"independent"`
	Dependent   []string `json:"dependent"`
	Controlled  []string `json:"controlled"`
}

// DiagramNode 结构图/流程图节点，参考技术方案 §16.4。
type DiagramNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"` // factor / result / process / structure
}

// DiagramEdge 结构图/流程图边。
type DiagramEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
}

// DiagramSpec 结构图/流程图协议，参考技术方案 §16.4。
type DiagramSpec struct {
	DiagramType string        `json:"diagram_type"` // flow / relation / hierarchy
	Title       string        `json:"title"`
	Nodes       []DiagramNode `json:"nodes"`
	Edges       []DiagramEdge `json:"edges"`
}

// BiologyModel 生物建模完整结果，参考技术方案 §16.3。
type BiologyModel struct {
	Topic               string               `json:"topic"`
	Concepts            []Concept            `json:"concepts"`
	Relations           []Relation           `json:"relations"`
	ProcessSteps        []ProcessStep        `json:"process_steps"`
	ExperimentVariables *ExperimentVariables `json:"experiment_variables,omitempty"`
	Diagram             *DiagramSpec         `json:"diagram,omitempty"`
	ResultSummary       string               `json:"result_summary"`
}
