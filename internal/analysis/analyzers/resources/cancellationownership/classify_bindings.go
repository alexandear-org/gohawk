package cancellationownership

import (
	"go/token"
	"slices"

	"github.com/kojah/gohawk/internal/engine/lifecycle"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"golang.org/x/tools/go/ssa"
)

// The generated classifiers select labels over these existing evidence queries.
// Owner discovery remains cached, and exact completion transports the same
// request (including its spent allowance) into the optional summary fallback.
type classifySignature func(ssa.Instruction, *cancellationClassifier) cancellationLabel

type callLabelSignature func(ssa.Instruction, *cancellationClassifier, *ssa.CallCommon) (cancellationLabel, bool)

type argumentLabelSignature func(ssa.Instruction, *cancellationClassifier) (cancellationLabel, bool)

type returnLabelSignature func(*ssa.Return, *cancellationClassifier) cancellationLabel

var (
	_ classifySignature      = classifyCancellation
	_ callLabelSignature     = recognizeCancellation
	_ callLabelSignature     = recognizeCancellationDirect
	_ callLabelSignature     = recognizeCancellationCall
	_ argumentLabelSignature = recognizeCancellationArgument
	_ argumentLabelSignature = recognizeCancellationDeferredUse
	_ returnLabelSignature   = classifyCancellationReturn
)

const (
	receiveToken        = token.ARROW
	completionProven    = proofs.EvidenceProven
	completionDisproven = proofs.EvidenceDisproven
	completionUnknown   = proofs.EvidenceUnknown
)

func (classifier *cancellationClassifier) ownerHoldsInstruction(instruction ssa.Instruction) bool {
	owner := classifier.cancellationOwner()
	return owner.Owner != nil && owner.Owner.holds[instruction]
}

func (classifier *cancellationClassifier) ownerAvailable() bool {
	return classifier.cancellationOwner().Known()
}

func (classifier *cancellationClassifier) recognizedInstruction(instruction ssa.Instruction) (cancellationLabel, bool) {
	return classifier.recognizedAction(instruction, ssaflow.InstructionCall(instruction))
}

func (classifier *cancellationClassifier) processLifetimeReturn(instruction ssa.Instruction) bool {
	_, returned := instruction.(*ssa.Return)
	return returned && classifier.processLifetime
}

func (classifier *cancellationClassifier) argumentCompletion(instruction ssa.Instruction) (proofs.EvidenceState, lifecycle.CompletionRequest) {
	request := lifecycle.CompletionRequest{
		Instruction: instruction, Target: classifier.cancel, InvokeTarget: true,
		Budget: classifier.budget(),
	}
	completion := lifecycle.ProveCompletion(request)
	return completion.State, request
}

func testingCleanupCall(common *ssa.CallCommon, cancel ssa.Value) bool {
	return common != nil && ssacall.HasLibraryContract(common, ssacall.ContractTestingCleanup) && commonHasExactArgument(common, cancel)
}

// Timers and framework registrars do not guarantee that an installed callback
// runs. Recognizing their documented contract leaves cancellation unknown.
func registeredCleanupCall(instruction ssa.Instruction, common *ssa.CallCommon, cancel ssa.Value) bool {
	return common != nil && (ssacall.HasLibraryContract(common, ssacall.ContractAfterFunc) ||
		ssacall.HasLibraryContract(common, ssacall.ContractDeferredCleanup)) && instructionReferencesCancellation(instruction, cancel)
}

func (classifier *cancellationClassifier) exactCallArgument(instruction ssa.Instruction, common *ssa.CallCommon) (cancellationLabel, bool) {
	if common == nil || !commonHasExactArgument(common, classifier.cancel) {
		return cancellationLabel{}, false
	}
	return classifier.exactArgumentAction(instruction)
}

// A callback which captures cancel may be invoked, retained or discarded.
// Captured identity alone cannot establish release. Vekil supplies these
// callbacks to helpers that own their execution:
// https://github.com/sozercan/vekil/blob/842f12f7875143274378fcbb80d411295edf3d28/cmd/menubar/portal_linux_test.go#L210-L230
func capturedCancellationArgument(common *ssa.CallCommon, cancel ssa.Value) bool {
	return common != nil && slices.ContainsFunc(common.Args, func(argument ssa.Value) bool {
		_, closure := argument.(*ssa.MakeClosure)
		return closure && lifecycle.MayContainValue(argument, cancel)
	})
}

// Direct cancellation compares exact SSA identities; broad alias or containment
// evidence cannot establish invocation of this cancel function.
func exactCancelCall(common *ssa.CallCommon, cancel ssa.Value) bool {
	return common != nil && common.Value == cancel
}

func returnsCancel(returned *ssa.Return, cancel ssa.Value) bool {
	return slices.Contains(returned.Results, cancel)
}
