package cancellationownership

import (
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"golang.org/x/tools/go/ssa"
)

// A fresh standard context child also follows its exact parent's cancellation.
// Reuse the existing classifier for that alternate owner, but only as unknown:
// an opaque handoff of the parent is not proof of when its child is canceled.
// Signal registrations still require their own stop function; cancellation of
// their parent does not unregister them. Wrapped and merged parents stay opaque.
// https://github.com/crazy-max/diun/blob/269cb27295944aeacfe549d24ab7ac483e600aa9/internal/notif/apprise/client.go#L91-L94
func parentCancellationClassifier(call *ssa.Call, observer proofs.Observer) *cancellationClassifier {
	contract, ok := cancellationContractFor(call.Common())
	if !ok || contract.packagePath != "context" || len(call.Common().Args) == 0 {
		return nil
	}
	// Contexts captured by workers are held in local cells. Resolve the load
	// where this child is created, not at the later cancellation or return: the
	// same source variable may subsequently hold the child instead of its parent.
	// https://github.com/werf/nelm/blob/6393382d695e65d8d8f744cf590337fe62a83eef/pkg/action/release_install.go#L179-L190
	parentValue := call.Common().Args[0]
	if resolved := heapmodel.NewStorage(proofs.NewSearchBudget(cancellationCompletionBudget).Observed(observer)).Resolve(parentValue); resolved.Proven() {
		parentValue = resolved.Value
	}
	return classifyParentOrigin(parentValue, call, observer)
}

// Parent origin recognition uses the resolved acquisition-time value only.
// It never follows wrappers or merges and does not resolve storage again.
type parentOriginSignature func(ssa.Value, *ssa.Call, proofs.Observer) *cancellationClassifier

var _ parentOriginSignature = classifyParentOrigin

func sameParentFrame(constructor, child *ssa.Call) bool {
	return constructor.Parent() == child.Parent()
}

func parentCancelResult(constructor *ssa.Call) (ssa.Value, bool) {
	contract, known := cancellationContractFor(constructor.Common())
	if !known {
		return nil, false
	}
	cancel := ssacall.CallResult(constructor, contract.result)
	return cancel, cancel != nil
}

func newParentClassifier(cancel ssa.Value, observer proofs.Observer) *cancellationClassifier {
	return &cancellationClassifier{cancel: cancel, actions: make(map[ssa.Instruction]cancellationAction), observer: observer}
}
