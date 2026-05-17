package balancer

import (
	"testing"
)

func mustBalance(t *testing.T, s string) *Equation {
	t.Helper()
	eq, err := Parse(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	if err := Balance(eq); err != nil {
		t.Fatalf("balance %q: %v", s, err)
	}
	return eq
}

func coefs(species []SpeciesCount) []int {
	out := make([]int, len(species))
	for i, s := range species {
		out[i] = s.Coef
	}
	return out
}

func assertCoefs(t *testing.T, got []int, want ...int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len: got %v want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("idx %d: got %v want %v", i, got, want)
		}
	}
}

func TestParseFormula(t *testing.T) {
	cases := map[string]map[string]int{
		"H2O":       {"H": 2, "O": 1},
		"NaOH":      {"Na": 1, "O": 1, "H": 1},
		"H2SO4":     {"H": 2, "S": 1, "O": 4},
		"Ca(OH)2":   {"Ca": 1, "O": 2, "H": 2},
		"Al2(SO4)3": {"Al": 2, "S": 3, "O": 12},
		"K3PO4":     {"K": 3, "P": 1, "O": 4},
	}
	for f, want := range cases {
		got, err := parseFormula(f)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		for e, n := range want {
			if got[e] != n {
				t.Errorf("%s element %s: got %d want %d", f, e, got[e], n)
			}
		}
	}
}

func TestBalanceNeutralization(t *testing.T) {
	eq := mustBalance(t, "NaOH + H2SO4 -> Na2SO4 + H2O")
	assertCoefs(t, coefs(eq.Reactants), 2, 1)
	assertCoefs(t, coefs(eq.Products), 1, 2)
}

func TestBalanceCombustion(t *testing.T) {
	eq := mustBalance(t, "C3H8 + O2 -> CO2 + H2O")
	// C3H8 + 5 O2 -> 3 CO2 + 4 H2O
	assertCoefs(t, coefs(eq.Reactants), 1, 5)
	assertCoefs(t, coefs(eq.Products), 3, 4)
}

func TestBalanceElectrolysis(t *testing.T) {
	eq := mustBalance(t, "H2O -> H2 + O2")
	// 2 H2O -> 2 H2 + O2
	assertCoefs(t, coefs(eq.Reactants), 2)
	assertCoefs(t, coefs(eq.Products), 2, 1)
}

func TestBalanceDisplacement(t *testing.T) {
	eq := mustBalance(t, "Fe + CuSO4 -> FeSO4 + Cu")
	assertCoefs(t, coefs(eq.Reactants), 1, 1)
	assertCoefs(t, coefs(eq.Products), 1, 1)
}

func TestBalanceRedox(t *testing.T) {
	eq := mustBalance(t, "Al + O2 -> Al2O3")
	// 4 Al + 3 O2 -> 2 Al2O3
	assertCoefs(t, coefs(eq.Reactants), 4, 3)
	assertCoefs(t, coefs(eq.Products), 2)
}

func TestBalanceParenEquation(t *testing.T) {
	eq := mustBalance(t, "Ca(OH)2 + HCl -> CaCl2 + H2O")
	// Ca(OH)2 + 2 HCl -> CaCl2 + 2 H2O
	assertCoefs(t, coefs(eq.Reactants), 1, 2)
	assertCoefs(t, coefs(eq.Products), 1, 2)
}

func TestBalanceInvalid(t *testing.T) {
	// Two unrelated species + missing element on product side → non-conserving.
	eq, err := Parse("Na + Cl2 -> Br")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := Balance(eq); err == nil {
		t.Errorf("expected error on non-conserving equation")
	}
}

func TestParseMissingArrow(t *testing.T) {
	if _, err := Parse("Na + H2O"); err == nil {
		t.Errorf("expected parse error")
	}
}
