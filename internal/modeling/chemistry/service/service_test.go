package service

import (
	"context"
	"testing"

	"github.com/beihai0xff/snowy/internal/modeling/chemistry"
)

func TestAnalyzeNeutralization(t *testing.T) {
	svc := NewService()
	pkg, err := svc.AnalyzeReaction(context.Background(), "NaOH + H2SO4 -> Na2SO4 + H2O")
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if pkg.ReactionType != chemistry.ReactionNeutralization {
		t.Errorf("got %s want neutralization", pkg.ReactionType)
	}
	if len(pkg.Equation.Reactants) != 2 || pkg.Equation.Reactants[0].Coef != 2 {
		t.Errorf("bad equation %+v", pkg.Equation)
	}
}

func TestAnalyzeDisplacement(t *testing.T) {
	svc := NewService()
	pkg, err := svc.AnalyzeReaction(context.Background(), "Fe + CuSO4 -> FeSO4 + Cu")
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if pkg.ReactionType != chemistry.ReactionDisplacement {
		t.Errorf("got %s want displacement", pkg.ReactionType)
	}
}

func TestAnalyzeElectrolysis(t *testing.T) {
	svc := NewService()
	pkg, err := svc.AnalyzeReaction(context.Background(), "电解 H2O -> H2 + O2")
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if pkg.ReactionType != chemistry.ReactionElectrolysis {
		t.Errorf("got %s want electrolysis", pkg.ReactionType)
	}
	if pkg.Conditions.Energy != "通电" {
		t.Errorf("conditions: %+v", pkg.Conditions)
	}
}

func TestBalanceOnly(t *testing.T) {
	svc := NewService()
	eq, err := svc.Balance(context.Background(), "C3H8 + O2 -> CO2 + H2O")
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if eq.Reactants[1].Coef != 5 || eq.Products[0].Coef != 3 || eq.Products[1].Coef != 4 {
		t.Errorf("wrong balance: %+v", eq)
	}
}

func TestSpeciesLookup(t *testing.T) {
	svc := NewService()
	pkg, err := svc.AnalyzeReaction(context.Background(), "2H2O -> 2H2 + O2")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if len(pkg.Species) < 3 {
		t.Fatalf("species: %+v", pkg.Species)
	}
	for _, sp := range pkg.Species {
		if sp.Fallback2D {
			t.Errorf("expected lookup hit for %s", sp.Formula)
		}
	}
}
