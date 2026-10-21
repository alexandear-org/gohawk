package lifecycle

import (
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/lifecycle/completionpolicy"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"golang.org/x/tools/go/ssa"
)

// Callee resolution supplies a complete, ordered set of visible callback bodies.
// Every origin and its stable-storage lookup share the completion allowance;
// a truncated set cannot establish that all possible callees complete.

// resolveCallees returns every body the instruction may run, or ok=false when
// some path reaches a callee the analysis cannot see. A deferred callback may
// be resolved through the documented sync.OnceFunc contract; a callback
// invoked now may not, because an earlier invocation could already have
// consumed the wrapper.
func resolveCallees(instruction ssa.Instruction, budget *proofs.SearchBudget) ([]completionCallee, bool) {
	if !budget.Spend() {
		return nil, false
	}
	plan := completionpolicy.RecognizeLaunch(instruction)
	switch plan.Kind {
	case completionpolicy.CalleeCommon:
		return calleesOf(plan.Common, plan.Launch, instruction, plan.AllowOnceFunc, budget)
	case completionpolicy.CalleeCallback:
		return closureCallees(plan.Callback, plan.Launch, budget)
	case completionpolicy.CalleeAbsent:
		return nil, false
	}
	return nil, false
}

func calleesOf(
	common *ssa.CallCommon, launch launchKind, invocation ssa.Instruction, allowOnceFunc bool, budget *proofs.SearchBudget,
) ([]completionCallee, bool) {
	if !budget.Spend() {
		return nil, false
	}
	if closure, ok := common.Value.(*ssa.MakeClosure); ok {
		function, _ := closure.Fn.(*ssa.Function)
		return []completionCallee{{launch: launch, common: common, closure: closure, function: function}}, function != nil
	}
	if function := common.StaticCallee(); function != nil {
		return []completionCallee{{launch: launch, common: common, function: function}}, true
	}
	closures, ok := exactCallbacks(common.Value, invocation, allowOnceFunc, budget)
	if !ok {
		return nil, false
	}
	result := make([]completionCallee, 0, len(closures))
	for _, closure := range closures {
		if !budget.Spend() {
			return nil, false
		}
		function, _ := closure.Fn.(*ssa.Function)
		if function == nil {
			return nil, false
		}
		result = append(result, completionCallee{launch: launch, common: common, closure: closure, function: function})
	}
	return result, true
}

func closureCallees(value ssa.Value, launch launchKind, budget *proofs.SearchBudget) ([]completionCallee, bool) {
	if !budget.Spend() {
		return nil, false
	}
	closure, ok := value.(*ssa.MakeClosure)
	if !ok {
		return nil, false
	}
	function, _ := closure.Fn.(*ssa.Function)
	return []completionCallee{{launch: launch, closure: closure, function: function}}, function != nil
}

// exactCallbacks resolves a callback value to the function literals it may
// hold. Loads require one dominating store, phi edges must all resolve, and
// other call results are opaque.
func exactCallbacks(value ssa.Value, invocation ssa.Instruction, allowOnceFunc bool, budget *proofs.SearchBudget) ([]*ssa.MakeClosure, bool) {
	forms := ssaflow.TransparentChangeInterface | ssaflow.TransparentChangeType | ssaflow.TransparentConvert | ssaflow.TransparentMakeInterface
	var result []*ssa.MakeClosure
	var resolve func(ssaflow.ReachingWalk, ssa.Value) bool
	resolve = func(walk ssaflow.ReachingWalk, value ssa.Value) bool {
		plan := completionpolicy.RecognizeCallback(value, allowOnceFunc)
		switch plan.Kind {
		case completionpolicy.CallbackExact:
			result = append(result, plan.Closure)
			return true
		case completionpolicy.CallbackFollow:
			return walk.Every(plan.Source, resolve)
		case completionpolicy.CallbackStorage:
			if stored := heapmodel.NewStorage(budget).StableContent(plan.Source, invocation); stored.Proven() {
				return walk.Every(stored.Value, resolve)
			}
		case completionpolicy.CallbackAbsent:
		}
		return false
	}
	if !ssaflow.NewReachingWalk(forms).Within(budget).Every(value, resolve) {
		return nil, false
	}
	return result, len(result) > 0
}
