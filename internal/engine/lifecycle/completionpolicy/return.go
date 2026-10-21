package completionpolicy

import (
	"go/token"

	"github.com/kojah/gohawk/internal/engine/heapmodel"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"golang.org/x/tools/go/ssa"
)

// ReturnKind selects immediate return coverage or a conditional forwarding
// search. Irrelevant means the return contradicts the requested result outcome.
type ReturnKind uint8

const (
	// ReturnUncovered retains a relevant return without established completion.
	ReturnUncovered ReturnKind = iota
	// ReturnIrrelevant excludes this return from the requested outcome.
	ReturnIrrelevant
	// ReturnCovered accepts earlier completion after checking the outcome.
	ReturnCovered
	// ReturnForward asks the enclosing search to prove the forwarded callee.
	ReturnForward
)

// ReturnPlan preserves the forwarded call and its result index. It does not
// search that body or establish coverage of the caller's mapped locals.
type ReturnPlan struct {
	Kind   ReturnKind
	Call   *ssa.Call
	Result int
}

type (
	returnedSignature       func(*ssa.Return, ssacall.CallCondition, bool, *proofs.SearchBudget) ReturnPlan
	returnedValueSignature  func(ssa.Value, ssacall.Outcome, *proofs.SearchBudget) ssa.Value
	returnedResultSignature func(ssa.Value, bool) ReturnPlan
)

var (
	_ returnedSignature       = RecognizeReturn
	_ returnedValueSignature  = ResolveReturnValue
	_ returnedResultSignature = ReturnResult
)

const (
	returnLoadToken = token.MUL
	returnTrue      = ssacall.OutcomeTrue
	returnFalse     = ssacall.OutcomeFalse
)

func returnedValue(returned *ssa.Return, index int) (ssa.Value, bool) {
	if index >= len(returned.Results) {
		return nil, false
	}
	return returned.Results[index], true
}

func resolveReturn(value ssa.Value, budget *proofs.SearchBudget) (ssa.Value, bool) {
	resolved := heapmodel.NewStorage(budget).Resolve(value)
	return resolved.Value, resolved.Proven()
}

// excludesReturn requires a known contradiction. An opaque value is relevant;
// absence of a decoded literal cannot discard an uncovered return.
func excludesReturn(outcome ssacall.Outcome, value ssa.Value) bool {
	holds, known := ssacall.OutcomeOf(outcome, value)
	return known && !holds
}
