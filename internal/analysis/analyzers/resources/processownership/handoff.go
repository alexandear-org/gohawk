package processownership

import (
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/lifecycle"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	ssapath "github.com/kojah/gohawk/internal/engine/ssaflow/path"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// This file owns possible Wait participation. Positive captures, merged
// receivers and nonreturning waiters leave ownership uncertain; none proves
// exact reaping. Discovery and completion draw on the supplied allowance.

// Handoff rules select uncertainty from the same identity, return and
// completion evidence. Native queries keep exact proof payloads and budgets.
type waitHandoffSignature func(ssa.Instruction, ssa.Value, *proofs.SearchBudget) proofs.Proof

var _ waitHandoffSignature = provePossibleWaitHandoff

func handoffCall(instruction ssa.Instruction) (*ssa.CallCommon, bool) {
	common := ssaflow.InstructionCall(instruction)
	return common, common != nil
}

func handoffWait(common *ssa.CallCommon) bool {
	return ssacall.CallMatchesSymbol(common, syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "os/exec", Receiver: "Cmd", Name: "Wait"}))
}

func mergedWaitReceiver(common *ssa.CallCommon, command ssa.Value) bool {
	receiver := ssaflow.CallReceiver(common)
	_, merged := receiver.(*ssa.Phi)
	return merged && heapmodel.MayAlias(receiver, command) && !heapmodel.NewStorage(nil).Same(receiver, command).Proven()
}

func handoffBody(common *ssa.CallCommon) (*ssa.Function, bool) {
	callee, _ := ssacall.DirectCallee(common)
	return callee, callee != nil && len(callee.Blocks) != 0
}

func handoffStarted(instruction ssa.Instruction) bool {
	_, started := instruction.(*ssa.Go)
	return started
}

func handoffReturns(function *ssa.Function, budget *proofs.SearchBudget) proofs.Proof {
	return ssapath.ProveNormalReturnWithin(function.Blocks[0], nil, budget).Proof
}

func handoffCompletion(instruction ssa.Instruction, command ssa.Value, budget *proofs.SearchBudget) proofs.Proof {
	return lifecycle.ProveCompletion(lifecycle.CompletionRequest{
		Instruction: instruction, Target: command, Methods: []string{"Wait"},
		Coverage: lifecycle.CoverageAnywhere, Budget: budget,
	}).Proof
}

func processProofCutoff(proof proofs.Proof) bool {
	return proof.Reason == proofs.EvidenceBudgetExhausted
}

func missingHandoff() proofs.Proof {
	return proofs.Proof{State: proofs.EvidenceDisproven, Reason: proofs.EvidenceNotFound}
}

func uncertainHandoff(proof proofs.Proof) proofs.Proof {
	proof.State = proofs.EvidenceUnknown
	return proof
}

func mergedHandoff() proofs.Proof {
	return proofs.Proof{State: proofs.EvidenceUnknown, Reason: proofs.EvidenceUnknownPointee}
}

// Callback arguments retain their native census and first-unknown payload.
// The generated caller asks the callback value before starting this scan.
func handoffArguments(common *ssa.CallCommon, command ssa.Value, budget *proofs.SearchBudget) proofs.Proof {
	for _, argument := range common.Args {
		if result := provePossibleCallbackCapture(argument, command, budget); result.State == proofs.EvidenceUnknown {
			return result
		}
	}
	return missingHandoff()
}

// A possible capture makes Wait participation unknown. The shared query proves
// only the structural capture, never invocation, joining or exact reaping.
func provePossibleCallbackCapture(value, command ssa.Value, budget *proofs.SearchBudget) proofs.Proof {
	proof := lifecycle.ProvePossibleClosureCaptureWithin(value, command, budget)
	if proof.Proven() {
		proof.State = proofs.EvidenceUnknown
	}
	return proof
}
