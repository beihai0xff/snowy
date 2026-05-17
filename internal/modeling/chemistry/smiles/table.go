// Package smiles 内置高频小分子的简化 3D 几何表（直接给坐标，避免引入 SMILES 解析）。
//
// 命中：返回原子坐标 + 键 + CPK 颜色。
// 未命中：调用方需走 LLM 兜底，或前端 2D 球棍降级。
package smiles

// CPK 元素配色（简化 14 元素）。
var CPK = map[string]string{
	"H":  "#FFFFFF",
	"C":  "#1A1A1A",
	"N":  "#3050F8",
	"O":  "#FF0D0D",
	"F":  "#90E050",
	"Na": "#AB5CF2",
	"Mg": "#8AFF00",
	"Al": "#BFA6A6",
	"P":  "#FF8000",
	"S":  "#FFFF30",
	"Cl": "#1FF01F",
	"K":  "#8F40D4",
	"Ca": "#3DFF00",
	"Fe": "#E06633",
	"Cu": "#C88033",
	"Zn": "#7D80B0",
	"Br": "#A62929",
	"I":  "#940094",
}

// Atom 简化原子。
type Atom struct {
	Element string
	X, Y, Z float64
}

// Bond 简化键。
type Bond struct {
	A, B  int
	Order int
}

// Molecule 几何表条目。
type Molecule struct {
	Formula string
	Atoms   []Atom
	Bonds   []Bond
}

// Table 高频小分子查表（坐标 Å）。值经手工对齐：键长、键角符合典型几何。
var Table = map[string]Molecule{
	"H2O": {
		Formula: "H2O",
		Atoms: []Atom{
			{"O", 0, 0, 0},
			{"H", 0.96, 0, 0},
			{"H", -0.24, 0.93, 0},
		},
		Bonds: []Bond{{0, 1, 1}, {0, 2, 1}},
	},
	"H2": {
		Formula: "H2",
		Atoms:   []Atom{{"H", 0, 0, 0}, {"H", 0.74, 0, 0}},
		Bonds:   []Bond{{0, 1, 1}},
	},
	"O2": {
		Formula: "O2",
		Atoms:   []Atom{{"O", 0, 0, 0}, {"O", 1.21, 0, 0}},
		Bonds:   []Bond{{0, 1, 2}},
	},
	"N2": {
		Formula: "N2",
		Atoms:   []Atom{{"N", 0, 0, 0}, {"N", 1.10, 0, 0}},
		Bonds:   []Bond{{0, 1, 3}},
	},
	"CO2": {
		Formula: "CO2",
		Atoms: []Atom{
			{"C", 0, 0, 0},
			{"O", 1.16, 0, 0},
			{"O", -1.16, 0, 0},
		},
		Bonds: []Bond{{0, 1, 2}, {0, 2, 2}},
	},
	"CO": {
		Formula: "CO",
		Atoms:   []Atom{{"C", 0, 0, 0}, {"O", 1.13, 0, 0}},
		Bonds:   []Bond{{0, 1, 3}},
	},
	"CH4": {
		Formula: "CH4",
		Atoms: []Atom{
			{"C", 0, 0, 0},
			{"H", 0.63, 0.63, 0.63},
			{"H", -0.63, -0.63, 0.63},
			{"H", 0.63, -0.63, -0.63},
			{"H", -0.63, 0.63, -0.63},
		},
		Bonds: []Bond{{0, 1, 1}, {0, 2, 1}, {0, 3, 1}, {0, 4, 1}},
	},
	"NH3": {
		Formula: "NH3",
		Atoms: []Atom{
			{"N", 0, 0, 0},
			{"H", 0.93, 0.32, 0},
			{"H", -0.47, 0.32, 0.81},
			{"H", -0.47, 0.32, -0.81},
		},
		Bonds: []Bond{{0, 1, 1}, {0, 2, 1}, {0, 3, 1}},
	},
	"HCl": {
		Formula: "HCl",
		Atoms:   []Atom{{"H", 0, 0, 0}, {"Cl", 1.27, 0, 0}},
		Bonds:   []Bond{{0, 1, 1}},
	},
	"NaOH": {
		Formula: "NaOH",
		Atoms: []Atom{
			{"Na", 0, 0, 0},
			{"O", 1.95, 0, 0},
			{"H", 2.91, 0, 0},
		},
		Bonds: []Bond{{0, 1, 1}, {1, 2, 1}},
	},
	"NaCl": {
		Formula: "NaCl",
		Atoms:   []Atom{{"Na", 0, 0, 0}, {"Cl", 2.36, 0, 0}},
		Bonds:   []Bond{{0, 1, 1}},
	},
	"H2SO4": {
		Formula: "H2SO4",
		Atoms: []Atom{
			{"S", 0, 0, 0},
			{"O", 1.43, 0, 0},
			{"O", -1.43, 0, 0},
			{"O", 0, 1.43, 0},
			{"O", 0, -1.43, 0},
			{"H", 2.40, 0, 0},
			{"H", -2.40, 0, 0},
		},
		Bonds: []Bond{{0, 1, 1}, {0, 2, 1}, {0, 3, 2}, {0, 4, 2}, {1, 5, 1}, {2, 6, 1}},
	},
	"Na2SO4": {
		Formula: "Na2SO4",
		Atoms: []Atom{
			{"S", 0, 0, 0},
			{"O", 1.49, 0, 0},
			{"O", -1.49, 0, 0},
			{"O", 0, 1.49, 0},
			{"O", 0, -1.49, 0},
			{"Na", 3.0, 0, 0},
			{"Na", -3.0, 0, 0},
		},
		Bonds: []Bond{{0, 1, 2}, {0, 2, 2}, {0, 3, 1}, {0, 4, 1}, {3, 5, 1}, {4, 6, 1}},
	},
	"Fe": {Formula: "Fe", Atoms: []Atom{{"Fe", 0, 0, 0}}},
	"Cu": {Formula: "Cu", Atoms: []Atom{{"Cu", 0, 0, 0}}},
	"Zn": {Formula: "Zn", Atoms: []Atom{{"Zn", 0, 0, 0}}},
	"Al": {Formula: "Al", Atoms: []Atom{{"Al", 0, 0, 0}}},
	"Na": {Formula: "Na", Atoms: []Atom{{"Na", 0, 0, 0}}},
	"K":  {Formula: "K", Atoms: []Atom{{"K", 0, 0, 0}}},
	"Cl2": {
		Formula: "Cl2",
		Atoms:   []Atom{{"Cl", 0, 0, 0}, {"Cl", 1.99, 0, 0}},
		Bonds:   []Bond{{0, 1, 1}},
	},
	"CuSO4": {
		Formula: "CuSO4",
		Atoms: []Atom{
			{"S", 0, 0, 0},
			{"O", 1.49, 0, 0},
			{"O", -1.49, 0, 0},
			{"O", 0, 1.49, 0},
			{"O", 0, -1.49, 0},
			{"Cu", 3.0, 0, 0},
		},
		Bonds: []Bond{{0, 1, 2}, {0, 2, 2}, {0, 3, 1}, {0, 4, 1}, {3, 5, 1}},
	},
	"FeSO4": {
		Formula: "FeSO4",
		Atoms: []Atom{
			{"S", 0, 0, 0},
			{"O", 1.49, 0, 0},
			{"O", -1.49, 0, 0},
			{"O", 0, 1.49, 0},
			{"O", 0, -1.49, 0},
			{"Fe", 3.0, 0, 0},
		},
		Bonds: []Bond{{0, 1, 2}, {0, 2, 2}, {0, 3, 1}, {0, 4, 1}, {3, 5, 1}},
	},
	"CaCl2": {
		Formula: "CaCl2",
		Atoms: []Atom{
			{"Ca", 0, 0, 0},
			{"Cl", 2.36, 0, 0},
			{"Cl", -2.36, 0, 0},
		},
		Bonds: []Bond{{0, 1, 1}, {0, 2, 1}},
	},
	"Ca(OH)2": {
		Formula: "Ca(OH)2",
		Atoms: []Atom{
			{"Ca", 0, 0, 0},
			{"O", 1.95, 0, 0},
			{"O", -1.95, 0, 0},
			{"H", 2.91, 0, 0},
			{"H", -2.91, 0, 0},
		},
		Bonds: []Bond{{0, 1, 1}, {0, 2, 1}, {1, 3, 1}, {2, 4, 1}},
	},
}

// Lookup 命中返回 (mol, true)；未命中返回 (zero, false)。
func Lookup(formula string) (Molecule, bool) {
	m, ok := Table[formula]
	return m, ok
}
