package lifecycle

import (
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/lifecycle/completionmapping"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"golang.org/x/tools/go/ssa"
)

// Mapping the caller's target onto a callee's locals. Each mapped local
// says how the callee reaches the target: exactly, as a projection of it,
// as an owner of it, or as a callback bound to it. The mapping is where the
// completion search meets the storage model. completionmapping owns the
// ordered rules; this boundary supplies request-local evidence and memo effects.

func (search *completionSearch) mappedLocals(callee completionCallee, target ssa.Value, invocation ssa.Instruction) []mappedLocal {
	var result []mappedLocal
	for binding := range ssacall.CallBindingsWithin(callee.common, callee.function, callee.closure, search.budget) {
		var local mappedLocal
		var ok bool
		if binding.Captured {
			local, ok = search.capturedLocal(callee, binding.Local, binding.Supplied, target, invocation)
		} else {
			local, ok = search.argumentLocal(binding.Local, binding.Supplied, target, invocation)
		}
		if search.budget.Exhausted() || search.budget.PoolExhausted() {
			return nil
		}
		if ok {
			result = append(result, local)
		}
	}
	// A completed prefix says nothing about an unvisited binding. Never let
	// metadata or mapping cutoff publish locals to the coverage proof.
	if search.budget.Exhausted() || search.budget.PoolExhausted() {
		return nil
	}
	return result
}

// capturedLocal maps captures using their observation-time evidence. Deferred
// contexts retain the original invocation test before their storage queries.
func (search *completionSearch) capturedLocal(
	callee completionCallee, free, binding, target ssa.Value, invocation ssa.Instruction,
) (mappedLocal, bool) {
	local, ok := completionmapping.Captured(search.mappingContext(callee.launch == launchDeferred && invocation != nil), free, binding, target, invocation)
	return mappedLocal(local), ok
}

func (search *completionSearch) argumentLocal(parameter, argument, target ssa.Value, invocation ssa.Instruction) (mappedLocal, bool) {
	local, ok := completionmapping.Argument(search.mappingContext(false), parameter, argument, target, invocation)
	return mappedLocal(local), ok
}

func (search *completionSearch) mappingContext(deferred bool) completionmapping.Context {
	return completionmapping.Context{Queries: mappingQueries{search}, Budget: search.budget, Exact: search.exactTarget, Deferred: deferred}
}

// mappingQueries delegates only search-owned evidence and availability effects.
// The rules determine which query is reached; these methods do not retry policy.
type mappingQueries struct{ search *completionSearch }

func (queries mappingQueries) Callback(value, target ssa.Value) bool {
	return queries.search.valueCallsMethod(value, target)
}

func (queries mappingQueries) MayContain(aggregate, target ssa.Value) bool {
	return ProveMayContainValueWithin(aggregate, target, queries.search.budget).Proven()
}

func (queries mappingQueries) DirectStore(cell ssa.Value) bool {
	return valueHasDirectStore(cell, queries.search.budget)
}

func (queries mappingQueries) DeferredValue(binding, target ssa.Value, invocation ssa.Instruction) heapmodel.StoredValue {
	return queries.search.deferredBindingValue(binding, target, invocation)
}

func (queries mappingQueries) StoredOnPath(cell, target ssa.Value, invocation ssa.Instruction) bool {
	return targetStoredOnPath(cell, target, invocation, queries.search.budget).Proven()
}

func (queries mappingQueries) Uncertain() (completionmapping.Local, bool) {
	*queries.search.incomplete = true
	return completionmapping.Local{}, false
}

func (queries mappingQueries) Cutoff() (completionmapping.Local, bool) {
	queries.search.memo.Incomplete()
	*queries.search.incomplete = true
	return completionmapping.Local{}, false
}

type receiverProof = completionmapping.ReceiverProof

func (local mappedLocal) receives(receiver, target ssa.Value, budget *proofs.SearchBudget) receiverProof {
	return completionmapping.Receives(completionmapping.Local(local), receiver, target, budget)
}

func receiverMatchProof(matches bool) receiverProof { return completionmapping.MatchProof(matches) }

// deferredBindingValue composes storage availability with completion's memo.
// An independent storage child cutoff cannot become a cached negative mapping.
func (search *completionSearch) deferredBindingValue(binding, target ssa.Value, invocation ssa.Instruction) heapmodel.StoredValue {
	stored := deferredBindingValue(binding, target, invocation, search.budget)
	if stored.Reason == proofs.EvidenceBudgetExhausted {
		search.memo.Incomplete()
		*search.incomplete = true
	}
	return stored
}

// requiresNestedPath identifies mappings where completing the local as a whole
// is insufficient: the nested answer must name the caller's exact field.
func (local mappedLocal) requiresNestedPath() bool {
	return completionmapping.Local(local).RequiresNestedPath()
}

// receiverPath names the path beneath the target of a completing call's
// receiver, which receives established stands for the target through local.
func (search *completionSearch) receiverPath(local mappedLocal, receiver, target ssa.Value) (string, bool) {
	if search.exactTarget {
		return search.mappedPath(local, target, nil, true)
	}
	actual, ok := heapmodel.AccessPathFromParameterWithin(receiver, local.Local, search.budget)
	return search.mappedPath(local, target, actual, ok)
}

// mappedPath translates a path beneath the local, on which a completing
// call was made, onto the target the local stands for. The local's own
// relation to the target decides the translation, and any relation the
// search cannot express as a static path leaves the result unknown.
func (search *completionSearch) mappedPath(local mappedLocal, target ssa.Value, actual []string, known bool) (string, bool) {
	return completionmapping.TranslatePath(completionmapping.Local(local), target, actual, known, search.budget)
}

func (search *completionSearch) mappedCompletionPath(local mappedLocal, target ssa.Value, paths completionPaths) (string, bool) {
	return search.mappedPath(local, target, ssaflow.SplitAccessPath(paths.Path), paths.Known())
}
