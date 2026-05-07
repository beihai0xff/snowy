package generative

import (
	"fmt"
	"strings"
)

// Validator checks generated model packages before they are returned/persisted.
type Validator interface {
	Validate(pkg *GenerativeModelPackage) ModelValidationReport
}

type DefaultValidator struct{}

func NewDefaultValidator() Validator { return &DefaultValidator{} }

func (v *DefaultValidator) Validate(pkg *GenerativeModelPackage) ModelValidationReport {
	report := ModelValidationReport{
		SchemaValid:   true,
		EvidenceValid: len(pkg.EvidenceRefs) > 0,
		DomainValid:   true,
		SafetyValid:   true,
		Confidence:    pkg.Confidence,
	}
	checks := []ValidationCheck{}
	fail := func(name, msg string) {
		checks = append(checks, ValidationCheck{Name: name, Status: "fail", Message: msg})
		report.DomainValid = false
	}
	pass := func(name string) {
		checks = append(checks, ValidationCheck{Name: name, Status: "pass"})
	}

	if strings.TrimSpace(pkg.Question) == "" {
		report.SchemaValid = false

		fail("question", "question is empty")
	}

	if strings.TrimSpace(pkg.Domain) == "" {
		report.SchemaValid = false

		fail("domain", "domain is empty")
	}

	if strings.TrimSpace(pkg.LearningModel.LearningGoal) == "" {
		report.SchemaValid = false

		fail("learning_goal", "learning goal is empty")
	}

	switch pkg.Domain {
	case DomainPhysics:
		validatePhysics(pkg, fail, pass)
	case DomainBiology:
		validateBiology(pkg, fail, pass)
	default:
		fail("domain_supported", fmt.Sprintf("unsupported domain %q", pkg.Domain))
	}

	if report.EvidenceValid {
		pass("evidence")
	} else {
		checks = append(checks, ValidationCheck{Name: "evidence", Status: "warn", Message: "no evidence refs attached"})
	}

	report.Checks = checks

	report.FallbackRequired = !report.SchemaValid || !report.DomainValid || !report.SafetyValid
	if report.FallbackRequired && report.FallbackReason == "" {
		report.FallbackReason = "generated package did not pass validation"
	}

	if report.Confidence <= 0 {
		report.Confidence = packageConfidence(pkg, report)
	}

	return report
}

func validatePhysics(pkg *GenerativeModelPackage, fail func(string, string), pass func(string)) {
	if pkg.SimulationLogic == nil {
		fail("simulation_logic", "physics package requires simulation_logic")

		return
	}

	if len(pkg.SimulationLogic.Formulas) == 0 {
		fail("formula_scope", "physics simulation requires at least one formula")
	} else {
		ok := true

		for _, formula := range pkg.SimulationLogic.Formulas {
			if strings.TrimSpace(formula.Expr) == "" {
				ok = false
			}
		}

		if ok {
			pass("formula_scope")
		} else {
			fail("formula_scope", "formula expression is empty")
		}
	}

	variables := pkg.SimulationLogic.Variables
	if len(variables) == 0 {
		variables = pkg.GenerativeModel.Variables
	}

	if len(variables) == 0 {
		fail("variables", "physics simulation requires variables")
	} else {
		ok := true

		for _, variable := range variables {
			if strings.TrimSpace(variable.Name) == "" || variable.Max < variable.Min {
				ok = false
			}
		}

		if ok {
			pass("variables")
		} else {
			fail("variables", "variable name/range is invalid")
		}
	}

	pass("grade_boundary")
}

func validateBiology(pkg *GenerativeModelPackage, fail func(string, string), pass func(string)) {
	if pkg.VisualizationGraph == nil {
		fail("visualization_graph", "biology package requires visualization_graph")

		return
	}

	if len(pkg.VisualizationGraph.Nodes) == 0 {
		fail("concept_nodes", "biology visualization requires nodes")
	} else {
		pass("concept_nodes")
	}

	nodes := map[string]struct{}{}
	for _, node := range pkg.VisualizationGraph.Nodes {
		nodes[node.ID] = struct{}{}
	}

	okEdges := true

	for _, edge := range pkg.VisualizationGraph.Edges {
		if _, ok := nodes[edge.Source]; !ok {
			okEdges = false
		}

		if _, ok := nodes[edge.Target]; !ok {
			okEdges = false
		}
	}

	if okEdges {
		pass("relation_edges")
	} else {
		fail("relation_edges", "edge references missing node")
	}

	if vars := pkg.VisualizationGraph.ExperimentVariables; vars != nil &&
		(len(vars.Independent) > 0 || len(vars.Dependent) > 0 || len(vars.Controlled) > 0) {
		pass("experiment_variables")
	} else {
		fail("experiment_variables", "biology package requires experiment variables")
	}

	pass("grade_boundary")
}

func packageConfidence(pkg *GenerativeModelPackage, report ModelValidationReport) float64 {
	confidence := pkg.Confidence
	if confidence <= 0 {
		confidence = 0.78
	}

	if !report.EvidenceValid {
		confidence -= 0.18
	}

	if !report.DomainValid {
		confidence -= 0.25
	}

	if confidence < 0.05 {
		return 0.05
	}

	if confidence > 0.99 {
		return 0.99
	}

	return confidence
}
