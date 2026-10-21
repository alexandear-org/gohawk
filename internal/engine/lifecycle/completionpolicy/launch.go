package completionpolicy

import (
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// LaunchKind is when an instruction runs a callee. Callback describes a body
// examined independently; merely creating or storing it proves no invocation.
type LaunchKind uint8

const (
	LaunchNone LaunchKind = iota
	// LaunchDeferred covers defer and testing Cleanup registrations.
	LaunchDeferred
	// LaunchCalled returns synchronously to its parent.
	LaunchCalled
	// LaunchStarted covers go statements and sync.WaitGroup.Go.
	LaunchStarted
	// LaunchCallback examines a body independently of its launch.
	LaunchCallback
)

// Reason retains the completion evidence associated with this launch form.
func (launch LaunchKind) Reason() proofs.EvidenceReason {
	switch launch {
	case LaunchDeferred:
		return proofs.EvidenceDeferredCompletion
	case LaunchCalled:
		return proofs.EvidenceCalledCompletion
	case LaunchStarted:
		return proofs.EvidenceStartedCompletion
	case LaunchCallback:
		return proofs.EvidenceCallbackCompletion
	case LaunchNone:
	}
	return proofs.EvidenceNone
}

// CalleeKind selects ordinary callee resolution or an exact callback literal.
type CalleeKind uint8

const (
	CalleeAbsent CalleeKind = iota
	CalleeCommon
	CalleeCallback
)

// Callee retains the original call/callback and its timing contract. Resolution,
// storage evidence and body coverage remain the caller's bounded searches.
type Callee struct {
	Kind          CalleeKind
	Launch        LaunchKind
	Common        *ssa.CallCommon
	Callback      ssa.Value
	AllowOnceFunc bool
}
type launchSignature func(ssa.Instruction) Callee

var _ launchSignature = RecognizeLaunch

var waitGroupGo = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "WaitGroup", Name: "Go"})

func groupCallback(common ssa.CallCommon) (ssa.Value, bool) {
	if !ssacall.CallMatchesSymbol(&common, waitGroupGo) || len(common.Args) != 2 {
		return nil, false
	}
	return common.Args[1], true
}

func cleanupCallback(common ssa.CallCommon) (ssa.Value, bool) {
	if !ssacall.HasLibraryContract(&common, ssacall.ContractTestingCleanup) || len(common.Args) == 0 {
		return nil, false
	}
	return common.Args[len(common.Args)-1], true
}
func commonOf(instruction ssa.CallInstruction) *ssa.CallCommon { return instruction.Common() }
