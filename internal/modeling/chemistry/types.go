// Package chemistry implements the v7 chemistry modeling domain.
//
// 数据契约对齐 docs/snowy-v7-conversational-modeling-platform.md §5.1。
// 输出 ChemistryReactionPackage 由 generative.CompilerService 在 domain=chemistry
// 分支注入到 GenerativeModelPackage.SimulationLogic 中（与 physics/biology 平行）。
package chemistry

// ReactionType 反应分类，与文档 §5.1 对齐。
type ReactionType string

const (
	ReactionNeutralization ReactionType = "neutralization"
	ReactionRedox          ReactionType = "redox"
	ReactionDisplacement   ReactionType = "displacement"
	ReactionMetathesis     ReactionType = "metathesis" // 复分解
	ReactionElectrolysis   ReactionType = "electrolysis"
	ReactionIonization     ReactionType = "ionization"
	ReactionHydrolysis     ReactionType = "hydrolysis"
	ReactionCombustion     ReactionType = "combustion"
	ReactionUnknown        ReactionType = "unknown"
)

// SpeciesCoef 化学计量数 + 物种。
type SpeciesCoef struct {
	Species string `json:"species"`
	Coef    int    `json:"coef"`
}

// BalancedEquation 配平后的化学方程式。
type BalancedEquation struct {
	Reactants []SpeciesCoef `json:"reactants"`
	Products  []SpeciesCoef `json:"products"`
	Arrow     string        `json:"arrow"` // "→" | "⇌" | "↑" | "↓"
}

// Conditions 反应条件。
type Conditions struct {
	Temperature string `json:"temperature,omitempty"` // "常温" / "加热" / "高温"
	Catalyst    string `json:"catalyst,omitempty"`
	Solvent     string `json:"solvent,omitempty"` // "水溶液" / "气相" / ...
	Energy      string `json:"energy,omitempty"`  // "通电" / "Δ" / "光照"
	Pressure    string `json:"pressure,omitempty"`
}

// ETransfer 氧化还原电子转移。
type ETransfer struct {
	FromElement string `json:"from_element"`
	ToElement   string `json:"to_element"`
	Electrons   int    `json:"electrons"`
	FromOx      int    `json:"from_oxidation"`
	ToOx        int    `json:"to_oxidation"`
}

// EnergyProfile 反应能量曲线（定性）。
type EnergyProfile struct {
	ReactantEnergy float64 `json:"reactant_energy"`
	ProductEnergy  float64 `json:"product_energy"`
	ActivationE    float64 `json:"activation_energy"`
	Exothermic     bool    `json:"exothermic"`
}

// AtomXYZ 原子的 3D 坐标，单位 Å。
type AtomXYZ struct {
	Element string  `json:"element"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Z       float64 `json:"z"`
}

// Bond 键。
type Bond struct {
	A     int `json:"a"`     // 原子 A 在 SpeciesModel.Atoms 的下标
	B     int `json:"b"`     // 原子 B 下标
	Order int `json:"order"` // 1 / 2 / 3
}

// SpeciesModel 单个分子的几何模型。
type SpeciesModel struct {
	Formula string    `json:"formula"`
	Atoms   []AtomXYZ `json:"atoms,omitempty"`
	Bonds   []Bond    `json:"bonds,omitempty"`
	Color   string    `json:"color,omitempty"`
	// Fallback2D 表示坐标未命中内置表，由前端用 2D 球棍兜底。
	Fallback2D bool `json:"fallback_2d,omitempty"`
}

// AnimationFrame 动画脚本帧。
type AnimationFrame struct {
	T         float64  `json:"t"`                   // 0..1 归一化时间
	Note      string   `json:"note"`                // "碰撞" / "键断" / "键成" / ...
	Highlight []string `json:"highlight,omitempty"` // 物种 formula 或 element
}

// AnimationScript 动画脚本。
type AnimationScript struct {
	Duration float64          `json:"duration"` // 秒
	Frames   []AnimationFrame `json:"frames"`
}

// InteractionControl 演示卡可调参数（变量名 + 控件类型）。
type InteractionControl struct {
	Variable string  `json:"variable"`
	Label    string  `json:"label"`
	Unit     string  `json:"unit,omitempty"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Default  float64 `json:"default"`
	Step     float64 `json:"step,omitempty"`
}

// InteractionPlan 交互方案。
type InteractionPlan struct {
	Controls []InteractionControl `json:"controls,omitempty"`
}

// ChemistryReactionPackage 化学反应建模包（v7 §5.1）。
type ChemistryReactionPackage struct {
	ReactionType     ReactionType     `json:"reaction_type"`
	Equation         BalancedEquation `json:"equation"`
	RawInput         string           `json:"raw_input,omitempty"`
	Conditions       Conditions       `json:"conditions"`
	ElectronTransfer []ETransfer      `json:"electron_transfer,omitempty"`
	EnergyProfile    *EnergyProfile   `json:"energy_profile,omitempty"`
	Species          []SpeciesModel   `json:"species,omitempty"`
	Animation        AnimationScript  `json:"animation"`
	InteractionPlan  InteractionPlan  `json:"interaction_plan"`
	Warnings         []string         `json:"warnings,omitempty"`
}
