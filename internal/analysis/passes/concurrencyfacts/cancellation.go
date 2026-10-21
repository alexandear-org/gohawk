package concurrencyfacts

// Cancellation is a request on an exact context signal, not a channel close or
// a worker join. Symbolic calls carry binding requirements: Context and
// CancelFunc types alone do not rule out custom implementations or callbacks.

import (
	"slices"

	"github.com/kojah/gohawk/internal/analysis/passes/concurrencyfacts/recognition"

	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"golang.org/x/tools/go/ssa"
)

func (engine *Engine) cancellationCall(call ssa.CallInstruction) (Summary, bool) {
	kind, value := recognition.ContextCall(call.Common())
	switch kind {
	case recognition.ContextEmpty:
		return Summary{}, true
	case recognition.ContextFresh:
		// Restrict fresh signals to non-canceling, known parents for now.
		// Timers, parent propagation, custom Context hooks, and AfterFunc
		// callbacks must not disappear into an effect-free constructor.
		parent := engine.storage.Resolve(value)
		origin, ok := parent.Value.(*ssa.Call)
		if !parent.Proven() || !ok || !recognition.EmptyContext(origin.Common()) {
			return Summary{Reason: ReasonContextParentUnknown}, true
		}
		return Summary{}, true
	case recognition.ContextObserve, recognition.ContextRequest:
		return engine.cancellationOperation(call, value, kind == recognition.ContextRequest), true
	default:
		return Summary{}, false
	}
}

func (engine *Engine) cancellationOperation(call ssa.CallInstruction, value ssa.Value, cancel bool) Summary {
	resource, ok := engine.reference(value)
	if !ok || !resource.Cancellation {
		return Summary{Reason: ReasonContextIdentityUnknown}
	}
	result := Summary{CancellationInputs: []Reference{resource}}
	if cancel {
		result.Operations = []Operation{{Kind: Cancel, Resource: resource, Source: call.Pos(), Site: call.Pos()}}
	}
	return finishCancellation(result)
}

func (engine *Engine) cancellationReference(value ssa.Value) (Reference, bool) {
	kind, call, observed := recognition.ContextOrigin(value)
	switch kind {
	case recognition.OriginUnknown:
		return Reference{}, false
	case recognition.OriginObserved:
		return engine.reference(observed)
	case recognition.OriginFresh:
		if origin, _ := engine.cancellationCall(call); origin.Complete() {
			return Reference{Value: call, Cancellation: true}, true
		}
	}
	return Reference{}, false
}

func cancellationBound(reference Reference) bool {
	call, ok := reference.Value.(*ssa.Call)
	return reference.Cancellation && !reference.Indirect && ok && recognition.CancelConstructor(call.Common())
}

// Requirements stay attached to conditional select summaries too. They must
// be discharged before graph expansion, not just before linear consumption.
func finishCancellation(summary Summary) Summary {
	if summary.Reason != ReasonNone && summary.Reason != ReasonContextBindingRequired &&
		summary.Reason != ReasonCallbackBindingRequired && summary.Reason != ReasonReplicatedWorkers &&
		summary.Reason != ReasonSelectAlternatives {
		return summary
	}
	// Channel-valued helper arguments may become Done projections only at
	// binding time. Retain that requirement even without a local Done call.
	for _, op := range summary.Operations {
		if op.Resource.Cancellation {
			requireCancellation(&summary, []Reference{op.Resource})
		}
	}
	if summary.operationCount() > maxOperations {
		return Summary{Reason: ReasonSummaryLimit}
	}
	for _, input := range summary.CancellationInputs {
		if !cancellationBound(input) {
			if summary.Reason != ReasonSelectAlternatives {
				summary.Reason = ReasonContextBindingRequired
			}
			return summary
		}
	}
	if summary.Reason == ReasonContextBindingRequired {
		summary.Reason = ReasonNone
	}
	return finishReplicas(finishCallbacks(summary))
}

// CancellationBound reports whether every conditional cancellation contract
// has an exact standard-library origin. Expansion must check this separately
// because select summaries are intentionally not linearly complete.
func (summary Summary) CancellationBound() bool {
	for _, input := range summary.CancellationInputs {
		if !cancellationBound(input) {
			return false
		}
	}
	return true
}

func composableLinear(summary Summary) bool {
	return summary.Reason == ReasonNone || summary.Reason == ReasonContextBindingRequired ||
		summary.Reason == ReasonCallbackBindingRequired || summary.Reason == ReasonReplicatedWorkers
}

func requireCancellation(summary *Summary, inputs []Reference) {
	for _, input := range inputs {
		if !slices.Contains(summary.CancellationInputs, input) {
			summary.CancellationInputs = append(summary.CancellationInputs, input)
		}
	}
}

func (engine *Engine) bindCancellationInputs(
	inputs []Reference, bindings []ssacall.CallBinding, instruction ssa.CallInstruction,
) ([]Reference, Reason) {
	result := make([]Reference, 0, len(inputs))
	for _, input := range inputs {
		if !engine.budget.Spend() {
			return nil, ReasonBudgetExhausted
		}
		bound, ok := engine.bind(input, bindings, instruction)
		if !ok || !bound.Cancellation {
			return nil, ReasonContextBindingUnknown
		}
		result = append(result, bound)
	}
	return result, ReasonNone
}
