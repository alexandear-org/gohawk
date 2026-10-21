package deferinloop

import (
	"slices"

	"github.com/kojah/gohawk/internal/analysis/passes/lifecyclefacts"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	analysisTrace "github.com/kojah/gohawk/internal/reporting/trace"
	"golang.org/x/tools/go/ssa"
)

// The rules call the existing evidence functions directly. These small hooks
// express call presence and the obligation's cleanup contract; no query host,
// request, Boolean result or decision wrapper is needed.
type useSignature func(*lifecyclefacts.LifecycleEvidence, analysisTrace.Probe, ssa.Instruction, deferObligation) (resourceStatus, deferReason)

var _ useSignature = classifyValueUse

func callDetails(instruction ssa.Instruction) (bool, *ssa.CallCommon) {
	common := ssaflow.InstructionCall(instruction)
	return common != nil, common
}

func cleanupReceiver(obligation deferObligation, common *ssa.CallCommon) bool {
	return obligation.closedBy(ssaflow.CallReceiver(common))
}

func cleanupMethod(obligation deferObligation, common *ssa.CallCommon) bool {
	return slices.Contains(obligation.cleanup, ssaflow.CallName(common))
}
