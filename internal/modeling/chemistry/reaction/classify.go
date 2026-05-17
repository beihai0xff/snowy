// Package reaction 反应类型分类：规则优先 + 通用兜底。
//
// 当前只用规则；LLM 兜底由调用方（service）提供。
package reaction

import (
	"strings"

	"github.com/beihai0xff/snowy/internal/modeling/chemistry"
	"github.com/beihai0xff/snowy/internal/modeling/chemistry/balancer"
)

// Classify 基于已配平的方程式 + 原始输入文本判断反应类型。
func Classify(eq *balancer.Equation, raw string) chemistry.ReactionType {
	rawLower := strings.ToLower(raw)
	if strings.Contains(rawLower, "电解") || strings.Contains(rawLower, "electrolysis") {
		return chemistry.ReactionElectrolysis
	}
	if strings.Contains(rawLower, "燃烧") || strings.Contains(rawLower, "combust") {
		return chemistry.ReactionCombustion
	}
	if strings.Contains(rawLower, "水解") || strings.Contains(rawLower, "hydroly") {
		return chemistry.ReactionHydrolysis
	}
	if strings.Contains(rawLower, "电离") || strings.Contains(rawLower, "ioniz") {
		return chemistry.ReactionIonization
	}
	if eq == nil {
		return chemistry.ReactionUnknown
	}

	hasAcid := false
	hasBase := false
	hasSalt := false
	hasMetal := false
	hasH2O := false
	hasO2 := false
	hasSimpleSubstance := 0

	for _, side := range [][]balancer.SpeciesCount{eq.Reactants, eq.Products} {
		for _, sp := range side {
			f := sp.Formula
			if f == "H2O" {
				hasH2O = true
			}
			if f == "O2" {
				hasO2 = true
			}
			if isAcid(f) {
				hasAcid = true
			}
			if isBase(f) {
				hasBase = true
			}
			if isSalt(f) {
				hasSalt = true
			}
			if isMetal(f) {
				hasMetal = true
			}
			if isSimpleSubstance(sp) {
				hasSimpleSubstance++
			}
		}
	}

	switch {
	case hasAcid && hasBase && hasH2O:
		return chemistry.ReactionNeutralization
	case hasMetal && hasSalt:
		return chemistry.ReactionDisplacement
	case hasO2 && hasSimpleSubstance > 0:
		return chemistry.ReactionCombustion
	case hasSimpleSubstance >= 1 && transferOfOxidation(eq):
		return chemistry.ReactionRedox
	case hasAcid && hasSalt, hasBase && hasSalt, hasSalt && hasH2O:
		return chemistry.ReactionMetathesis
	}
	return chemistry.ReactionUnknown
}

func isAcid(f string) bool {
	// 简化：以 H 开头并含非金属离子
	if !strings.HasPrefix(f, "H") {
		return false
	}
	return strings.ContainsAny(f, "ClBrISFNO") && f != "H2O" && f != "H2"
}

func isBase(f string) bool {
	if !strings.HasSuffix(f, "OH") && !strings.HasSuffix(f, ")2") && !strings.HasSuffix(f, ")3") {
		return false
	}
	return strings.Contains(f, "OH")
}

func isSalt(f string) bool {
	// 启发式：含金属 + 非金属，但不是基/酸/水
	if isBase(f) || isAcid(f) || f == "H2O" {
		return false
	}
	hasMetal := false
	for _, m := range []string{"Na", "K", "Li", "Mg", "Ca", "Al", "Fe", "Cu", "Zn", "Ba", "Ag", "Pb"} {
		if strings.HasPrefix(f, m) {
			hasMetal = true
			break
		}
	}
	hasNonmetal := strings.ContainsAny(f, "ClSNOPI")
	return hasMetal && hasNonmetal
}

func isMetal(f string) bool {
	metals := []string{"Na", "K", "Li", "Mg", "Ca", "Al", "Fe", "Cu", "Zn", "Ba", "Ag", "Pb"}
	for _, m := range metals {
		if f == m {
			return true
		}
	}
	return false
}

func isSimpleSubstance(sp balancer.SpeciesCount) bool {
	return len(sp.Elements) == 1
}

func transferOfOxidation(eq *balancer.Equation) bool {
	// 启发式：反应物或产物有简单物质，且含金属 → 氧化还原。
	for _, side := range [][]balancer.SpeciesCount{eq.Reactants, eq.Products} {
		for _, sp := range side {
			if isSimpleSubstance(sp) && isMetal(sp.Formula) {
				return true
			}
		}
	}
	return false
}
