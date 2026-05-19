package generative

import (
	"context"
	"fmt"
)

//nolint:cyclop // Chemistry injection has distinct nil/error/defaulting branches.
func (s *compilerService) applyChemistryAnalysis(
	ctx context.Context,
	req *CompileRequest,
	pkg *GenerativeModelPackage,
	domain string,
) {
	if pkg == nil || domain != DomainChemistry || s.chemSvc == nil {
		return
	}

	result, err := s.chemSvc.AnalyzeReaction(ctx, req.Message)
	if err != nil || result == nil {
		if err != nil {
			pkg.Warnings = append(pkg.Warnings, fmt.Sprintf("chemistry analysis failed: %v", err))
		}

		return
	}

	if pkg.SimulationLogic == nil {
		pkg.SimulationLogic = &DynamicSimulationSpec{}
	}

	if pkg.SimulationLogic.SimulationType == "" {
		pkg.SimulationLogic.SimulationType = "chemistry_reaction"
	}

	if pkg.SimulationLogic.Runtime == "" {
		pkg.SimulationLogic.Runtime = "chemistry"
	}

	pkg.SimulationLogic.ChemistryReaction = result
	if !pkg.SimulationLogic.LocalRecomputeAllowed {
		pkg.SimulationLogic.LocalRecomputeAllowed = false
	}
}
