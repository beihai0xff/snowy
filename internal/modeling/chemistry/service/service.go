// Package service exposes the chemistry domain service used by the agent + generative compiler.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/beihai0xff/snowy/internal/modeling/chemistry"
	"github.com/beihai0xff/snowy/internal/modeling/chemistry/balancer"
	"github.com/beihai0xff/snowy/internal/modeling/chemistry/reaction"
	"github.com/beihai0xff/snowy/internal/modeling/chemistry/smiles"
)

// Service v7 §5：化学建模服务。
type Service interface {
	AnalyzeReaction(ctx context.Context, rawEquation string) (*chemistry.ChemistryReactionPackage, error)
	Balance(ctx context.Context, rawEquation string) (*chemistry.BalancedEquation, error)
}

type defaultService struct{}

// NewService 返回默认实现（纯规则 + 内置 SMILES 表）。
func NewService() Service { return &defaultService{} }

var (
	ErrInput   = errors.New("invalid input")
	ErrBalance = errors.New("balance failed")
)

func (s *defaultService) Balance(_ context.Context, raw string) (*chemistry.BalancedEquation, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("%w: empty", ErrInput)
	}
	eq, err := balancer.Parse(extractEquationPart(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInput, err)
	}
	if err := balancer.Balance(eq); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBalance, err)
	}
	return toContractEquation(eq), nil
}

func (s *defaultService) AnalyzeReaction(_ context.Context, raw string) (*chemistry.ChemistryReactionPackage, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("%w: empty", ErrInput)
	}
	cleaned := extractEquationPart(raw)
	eq, err := balancer.Parse(cleaned)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInput, err)
	}
	warnings := []string{}
	if err := balancer.Balance(eq); err != nil {
		warnings = append(warnings, fmt.Sprintf("balance failed: %v", err))
	}

	rtype := reaction.Classify(eq, raw)
	pkg := &chemistry.ChemistryReactionPackage{
		ReactionType:    rtype,
		Equation:        *toContractEquation(eq),
		RawInput:        raw,
		Conditions:      deriveConditions(raw, rtype),
		Species:         buildSpecies(eq),
		Animation:       buildAnimation(eq, rtype),
		InteractionPlan: chemistry.InteractionPlan{Controls: defaultControls(rtype)},
		Warnings:        warnings,
	}
	if rtype == chemistry.ReactionRedox || rtype == chemistry.ReactionElectrolysis || rtype == chemistry.ReactionCombustion {
		pkg.ElectronTransfer = deriveElectronTransfer(eq)
		pkg.EnergyProfile = &chemistry.EnergyProfile{
			ReactantEnergy: 1.0,
			ProductEnergy:  0.4,
			ActivationE:    1.4,
			Exothermic:     true,
		}
	}
	return pkg, nil
}

func toContractEquation(eq *balancer.Equation) *chemistry.BalancedEquation {
	out := &chemistry.BalancedEquation{Arrow: normalizeArrow(eq.Arrow)}
	for _, sp := range eq.Reactants {
		out.Reactants = append(out.Reactants, chemistry.SpeciesCoef{Species: sp.Formula, Coef: sp.Coef})
	}
	for _, sp := range eq.Products {
		out.Products = append(out.Products, chemistry.SpeciesCoef{Species: sp.Formula, Coef: sp.Coef})
	}
	return out
}

func normalizeArrow(a string) string {
	switch a {
	case "->", "→", "=":
		return "→"
	case "⇌":
		return "⇌"
	}
	return "→"
}

// extractEquationPart 从用户原始输入中提取化学方程式部分：
// 从第一个 ASCII 大写字母（元素首字母）开始截取，丢弃前置中文/空白。
func extractEquationPart(raw string) string {
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c >= 'A' && c <= 'Z' {
			return strings.TrimSpace(raw[i:])
		}
	}
	return strings.TrimSpace(raw)
}

func deriveConditions(raw string, rtype chemistry.ReactionType) chemistry.Conditions {
	lower := strings.ToLower(raw)
	cond := chemistry.Conditions{Temperature: "常温", Solvent: "水溶液"}
	if strings.Contains(raw, "加热") || strings.Contains(lower, "δ") || strings.Contains(raw, "高温") {
		cond.Temperature = "加热"
		cond.Energy = "Δ"
	}
	if rtype == chemistry.ReactionElectrolysis {
		cond.Energy = "通电"
		cond.Solvent = "水溶液"
	}
	if rtype == chemistry.ReactionCombustion {
		cond.Solvent = "气相"
		cond.Energy = "点燃"
	}
	if strings.Contains(raw, "催化剂") || strings.Contains(lower, "catalyst") {
		cond.Catalyst = "见原文"
	}
	return cond
}

func buildSpecies(eq *balancer.Equation) []chemistry.SpeciesModel {
	uniq := map[string]bool{}
	out := []chemistry.SpeciesModel{}
	add := func(formula string) {
		if uniq[formula] {
			return
		}
		uniq[formula] = true
		mol, ok := smiles.Lookup(formula)
		if !ok {
			out = append(out, chemistry.SpeciesModel{Formula: formula, Fallback2D: true})
			return
		}
		sm := chemistry.SpeciesModel{Formula: formula}
		for _, a := range mol.Atoms {
			sm.Atoms = append(sm.Atoms, chemistry.AtomXYZ{Element: a.Element, X: a.X, Y: a.Y, Z: a.Z})
		}
		for _, b := range mol.Bonds {
			sm.Bonds = append(sm.Bonds, chemistry.Bond{A: b.A, B: b.B, Order: b.Order})
		}
		if len(mol.Atoms) > 0 {
			sm.Color = smiles.CPK[mol.Atoms[0].Element]
		}
		out = append(out, sm)
	}
	for _, sp := range eq.Reactants {
		add(sp.Formula)
	}
	for _, sp := range eq.Products {
		add(sp.Formula)
	}
	return out
}

func buildAnimation(eq *balancer.Equation, rtype chemistry.ReactionType) chemistry.AnimationScript {
	frames := []chemistry.AnimationFrame{
		{T: 0, Note: "反应物分离"},
		{T: 0.3, Note: "靠近碰撞"},
		{T: 0.55, Note: "键断裂"},
		{T: 0.75, Note: "键重组"},
		{T: 1.0, Note: "产物分离"},
	}
	if rtype == chemistry.ReactionElectrolysis {
		frames = []chemistry.AnimationFrame{
			{T: 0, Note: "电解质溶液"},
			{T: 0.35, Note: "通电，离子迁移"},
			{T: 0.7, Note: "阴/阳极析出"},
			{T: 1.0, Note: "产物聚集"},
		}
	}
	if eq != nil && len(eq.Reactants) > 0 && len(frames) > 0 {
		hl := make([]string, 0, len(eq.Reactants))
		for _, sp := range eq.Reactants {
			hl = append(hl, sp.Formula)
		}
		frames[0].Highlight = hl
	}
	return chemistry.AnimationScript{Duration: 4.0, Frames: frames}
}

func defaultControls(rtype chemistry.ReactionType) []chemistry.InteractionControl {
	c := []chemistry.InteractionControl{
		{Variable: "concentration", Label: "浓度", Unit: "mol/L", Min: 0.01, Max: 5.0, Default: 1.0, Step: 0.01},
		{Variable: "temperature", Label: "温度", Unit: "°C", Min: 0, Max: 200, Default: 25, Step: 1},
	}
	if rtype == chemistry.ReactionElectrolysis {
		c = append(c, chemistry.InteractionControl{Variable: "voltage", Label: "电压", Unit: "V", Min: 0.1, Max: 12, Default: 5, Step: 0.1})
	}
	if rtype == chemistry.ReactionCombustion {
		c = append(c, chemistry.InteractionControl{Variable: "oxygen_ratio", Label: "氧气比例", Min: 0.05, Max: 1.0, Default: 0.21, Step: 0.01})
	}
	return c
}

func deriveElectronTransfer(eq *balancer.Equation) []chemistry.ETransfer {
	if eq == nil {
		return nil
	}
	metals := []string{"Na", "K", "Mg", "Ca", "Al", "Fe", "Cu", "Zn"}
	isMetal := func(e string) bool {
		for _, m := range metals {
			if e == m {
				return true
			}
		}
		return false
	}
	var from, to string
	for _, sp := range eq.Reactants {
		if len(sp.Elements) == 1 {
			for e := range sp.Elements {
				if isMetal(e) {
					from = e
				}
			}
		}
	}
	for _, sp := range eq.Products {
		if len(sp.Elements) == 1 {
			for e := range sp.Elements {
				if isMetal(e) && e != from {
					to = e
				}
			}
		}
	}
	if from == "" || to == "" {
		return nil
	}
	return []chemistry.ETransfer{{FromElement: from, ToElement: to, Electrons: 2, FromOx: 0, ToOx: 2}}
}
