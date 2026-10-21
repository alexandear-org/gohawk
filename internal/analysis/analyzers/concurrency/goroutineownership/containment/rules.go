package containment

import (
	"go/token"
	"slices"

	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/lifecycle"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"golang.org/x/tools/go/ssa"
)

// Containment recognition selects existing recursive queries without taking
// ownership of their walk. Escape queries retain the caller's possible-value
// predicate and helper-search state; an escape answer never proves a join.
type (
	containmentSignature func(ssaflow.ReachingWalk, ssa.Value, ssa.Value) bool
	storedUseSignature   func(ssaflow.ReachingWalk, ssa.Instruction, ssa.Value, ssa.Value) bool
	escapeSignature      func(ssa.Instruction, ssa.Value, func(ssa.Value) bool, func(ssa.Instruction) bool) bool
)

var (
	_ containmentSignature = decideContainment
	_ storedUseSignature   = decideStoredUse
	_ escapeSignature      = Escapes
)

const loadToken = token.MUL

func closureCarries(walk ssaflow.ReachingWalk, closure *ssa.MakeClosure, target ssa.Value) bool {
	return slices.ContainsFunc(closure.Bindings, func(binding ssa.Value) bool {
		return heapmodel.CapturedBindingMatches(binding, target) || Matches(walk, binding, target)
	})
}

func sameValue(first, second ssa.Value) bool { return first == second }

func derived(matches func(ssa.Value) bool, value ssa.Value) bool { return matches(value) }

func captured(closure *ssa.MakeClosure, matches func(ssa.Value) bool) bool {
	return slices.ContainsFunc(closure.Bindings, matches)
}

func selectHandoff(choice *ssa.Select, local ssa.Value) bool {
	return slices.ContainsFunc(choice.States, func(state *ssa.SelectState) bool {
		return state.Send != nil && ssaflow.ValueIsAccessPathFrom(state.Chan, local)
	})
}

func returnedHandoff(returned *ssa.Return, local ssa.Value, matches func(ssa.Value) bool) bool {
	return lifecycle.ReturnedValueOwnsValue(returned, local) || slices.ContainsFunc(returned.Results, func(value ssa.Value) bool {
		return ssaflow.ChannelType(value) && matches(value)
	})
}

func called(matches func(ssa.Instruction) bool, instruction ssa.Instruction) bool {
	return matches(instruction)
}
