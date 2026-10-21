package concurrencyfacts

import (
	"go/types"
	"slices"

	"github.com/kojah/gohawk/internal/analysis/passes/concurrencyfacts/admission"
	"github.com/kojah/gohawk/internal/engine/ssaflow/calls/synchronization"

	"github.com/kojah/gohawk/internal/analysis/passes/concurrencyfacts/recognition"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"golang.org/x/tools/go/ssa"
)

func (engine *Engine) callSummary(instruction ssa.CallInstruction) Summary {
	if result, handled := engine.cancellationCall(instruction); handled {
		return result
	}
	if hole, ok := callbackHole(instruction); ok {
		return hole
	}
	common := engine.resolvedCommon(instruction)
	if recognition.ConditionConstructor(common) {
		if _, ok := condLocker(instruction); ok {
			return Summary{}
		}
		return Summary{Reason: ReasonCondLockerUnknown}
	}
	if operation, recognized := synchronization.Recognize(common); recognized {
		if operation.Kind == GroupAdd && (!operation.Fixed || operation.Count != 1) {
			return Summary{Reason: ReasonGroupCountUnknown}
		}
		var result Summary
		result.Reason = engine.appendOperation(&result, operation.Kind, operation.Resource, instruction.Pos())
		return result
	}
	if admission.Builtin(common) || admission.Atomic(common) {
		return Summary{}
	}
	return engine.instantiate(instruction)
}

func (engine *Engine) deferCompletion(result *Summary, instruction *ssa.Defer) Reason {
	called := engine.callSummary(instruction)
	// A deferred helper can launch another participant. Its synchronous
	// cleanup operations alone are not an exhaustive deferred effect list.
	if len(called.Workers) != 0 {
		return ReasonDeferredEffectsUnknown
	}
	if !composableLinear(called) {
		return called.Reason
	}
	if len(called.Operations) == 0 {
		return ReasonDeferredEffectsUnknown
	}
	for _, op := range called.Operations {
		if op.Kind != Close && op.Kind != GroupDone && op.Kind != Unlock && op.Kind != ReadUnlock && op.Kind != Cancel {
			return ReasonDeferredEffectsUnknown
		}
	}
	result.CancellationInputs = append(result.CancellationInputs, called.CancellationInputs...)
	// Arguments are bound now. Only their execution moves to RunDefers;
	// captured cells still require stable storage through the invocation.
	// RunDefers reverses this stack, so push the helper backwards to retain
	// its internal execution order while reversing the order of deferred calls.
	for _, op := range slices.Backward(called.Operations) {
		result.deferred = append(result.deferred, op)
	}
	return ReasonNone
}

// MutexPointer identifies sync.Mutex and sync.RWMutex pointers, not lookalikes.
func MutexPointer(value types.Type) bool { return admission.MutexPointer(value) }

func straightLineBody(function *ssa.Function) bool {
	if len(function.Blocks) == 1 {
		return true
	}
	// A function with defers also has a detached recovery block; see
	// detachedRecovery for why it is not an ordinary branch.
	return len(function.Blocks) == 2 && function.Recover == function.Blocks[1] &&
		len(function.Blocks[0].Succs) == 0 && detachedRecovery(function)
}

// Conditions retain their own identity in summaries. The associated locker
// can only be recovered from an exact NewCond call, never a mutable L field.

func condLocker(call ssa.CallInstruction) (ssa.Value, bool) {
	if !recognition.ConditionConstructor(call.Common()) || len(call.Common().Args) != 1 {
		return nil, false
	}
	wrapped, ok := call.Common().Args[0].(*ssa.MakeInterface)
	if !ok || !MutexPointer(wrapped.X.Type()) {
		return nil, false
	}
	return wrapped.X, true
}

// CondMutex resolves the exact Mutex supplied to a NewCond allocation. It does
// not prove that L remains unchanged: callers need a complete root summary,
// which rejects condition-field access, mutation and opaque publication.
func (engine *Engine) CondMutex(reference Reference, budget *proofs.SearchBudget) (Reference, bool) {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	call, ok := reference.Value.(*ssa.Call)
	if !ok || reference.Indirect {
		return Reference{}, false
	}
	locker, ok := condLocker(call)
	if !ok {
		return Reference{}, false
	}
	return engine.query(budget).reference(locker)
}
