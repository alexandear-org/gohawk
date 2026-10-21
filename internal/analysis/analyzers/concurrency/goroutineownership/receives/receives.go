// Package receives identifies receive sites through caller-supplied channel
// predicates. It keeps mixed select arms separate and does not decide whether
// channel identity establishes a join or only possible ownership.
package receives

import (
	"go/token"
	"go/types"
	"slices"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	ssapath "github.com/kojah/gohawk/internal/engine/ssaflow/path"
	"golang.org/x/tools/go/ssa"
)

// From reports whether instruction receives from a channel accepted by
// matches, through a receive expression, a select case, or a channel range.
func From(instruction ssa.Instruction, matches func(ssa.Value) bool) bool {
	return FromWithin(instruction, matches, nil)
}

// FromWithin bounds select-case enumeration with the supplied allowance.
func FromWithin(instruction ssa.Instruction, matches func(ssa.Value) bool, budget *proofs.SearchBudget) bool {
	return decideFrom(instruction, matches, budget)
}

// Guaranteed requires a receive on every selected path.
// A mixed select does not settle every path. Its exact receive arm is credited
// on the edge, or at an unambiguous arm entry, keeping other cases open.
// https://github.com/containerd/stargz-snapshotter/blob/624678b4e421947534cbf0618f9609853cccee0f/store/manager.go#L193-L221
func Guaranteed(instruction ssa.Instruction, matches func(ssa.Value) bool) bool {
	return decideGuaranteed(instruction, matches)
}

// SelectedAtEntry recognizes an unambiguous selected receive at an arm entry.
func SelectedAtEntry(instruction ssa.Instruction, matches func(ssa.Value) bool) bool {
	block := instruction.Block()
	if len(block.Instrs) == 0 || block.Instrs[0] != instruction {
		return false
	}
	channel, selected := ssapath.SelectedReceiveChannel(block)
	return selected && matches(channel)
}

// Native predicate parameters retain the caller's exact or possible channel
// policy. Invocations are queries because those predicates may spend budgets.
type (
	fromSignature       func(ssa.Instruction, func(ssa.Value) bool, *proofs.SearchBudget) bool
	guaranteedSignature func(ssa.Instruction, func(ssa.Value) bool) bool
)

var (
	_ fromSignature       = decideFrom
	_ guaranteedSignature = decideGuaranteed
)

const receiveToken = token.ARROW

func matchChannel(matches func(ssa.Value) bool, channel ssa.Value) bool { return matches(channel) }

func channelRange(value ssa.Value) bool {
	_, channel := value.Type().Underlying().(*types.Chan)
	return channel
}

func selectReceives(choice *ssa.Select, matches func(ssa.Value) bool, budget *proofs.SearchBudget) bool {
	for _, state := range choice.States {
		if !budget.Spend() {
			return false
		}
		if state.Dir == types.RecvOnly && matches(state.Chan) {
			return true
		}
	}
	return false
}

func selectGuaranteed(choice *ssa.Select, matches func(ssa.Value) bool) bool {
	return choice.Blocking && len(choice.States) > 0 && !slices.ContainsFunc(choice.States, func(state *ssa.SelectState) bool {
		return state.Dir != types.RecvOnly || !matches(state.Chan)
	})
}
