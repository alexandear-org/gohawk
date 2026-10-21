package calls

import (
	"go/token"

	ssaflow "github.com/kojah/gohawk/internal/engine/ssaflow"
	"golang.org/x/tools/go/ssa"
)

// Walk outward from a supplied value, not backwards through possible origins.
// Every alias use must be accounted for before the absence of an effect is a
// proof. Local spills deliberately count as retention until their aliases can
// be modeled; unsupported instructions never silently become read-only.
func (query *CallEffects) uses(value ssa.Value, seen map[ssa.Value]bool) CallEffect {
	if seen[value] {
		return 0
	}
	if value.Referrers() == nil {
		return effectUnknown
	}
	seen[value] = true
	var effects CallEffect
	for _, use := range *value.Referrers() {
		if !query.budget.Spend() {
			return effects | effectUnknown
		}
		effects |= query.use(value, use, seen)
	}
	return effects
}

// callUseSignature returns direct effects or one transparent alias to visit.
// The caller retains recursive dispatch and the shared use-walk state.
type callUseSignature func(value ssa.Value, use ssa.Instruction) (CallEffect, ssa.Value)

var _ callUseSignature = classifyCallUse

const (
	useLoadToken        = token.MUL
	useEqualToken       = token.EQL
	useNotEqualToken    = token.NEQ
	useStored           = EffectRetain | effectUnknown
	useStoredAndMutated = EffectMutate | useStored
	useSent             = EffectRetain | EffectAsync | effectUnknown
)

// SSA values use exact node identity here, without alias resolution.
func sameEffectValue(left, right ssa.Value) bool { return left == right }

// The rules call this only after identifying a value-producing alias form.
func effectAlias(instruction ssa.Instruction) ssa.Value {
	value, _ := instruction.(ssa.Value)
	return value
}

func (query *CallEffects) use(value ssa.Value, use ssa.Instruction, seen map[ssa.Value]bool) CallEffect {
	switch use := use.(type) {
	case *ssa.Call, *ssa.Defer, *ssa.Go:
		return query.call(use, value)
	case *ssa.MakeClosure:
		return query.closure(use, value)
	}
	effects, alias := classifyCallUse(value, use)
	if alias != nil {
		return query.uses(alias, seen)
	}
	return effects
}

func (query *CallEffects) closure(closure *ssa.MakeClosure, value ssa.Value) CallEffect {
	function, ok := closure.Fn.(*ssa.Function)
	if !ok || closure.Referrers() == nil {
		return effectUnknown
	}
	var effects CallEffect
	for binding := range CallBindingsWithin(nil, function, closure, query.budget) {
		if binding.Supplied == value {
			effects |= query.value(binding.Local)
		}
	}
	if !query.bindingMetadataComplete() {
		return effects | effectUnknown
	}
	for _, use := range *closure.Referrers() {
		if !query.budget.Spend() {
			return effects | effectUnknown
		}
		common := ssaflow.InstructionCall(use)
		if common == nil || common.Value != closure {
			effects |= EffectRetain | effectUnknown
			continue
		}
		if _, launched := use.(*ssa.Go); launched {
			effects |= EffectAsync
		}
	}
	return effects
}
