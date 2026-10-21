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
	switch typed := instruction.(type) {
	case *ssa.UnOp:
		return typed.Op == token.ARROW && matches(typed.X)
	case *ssa.Select:
		for _, state := range typed.States {
			if !budget.Spend() {
				return false
			}
			if state.Dir == types.RecvOnly && matches(state.Chan) {
				return true
			}
		}
	case *ssa.Range:
		_, channel := typed.X.Type().Underlying().(*types.Chan)
		return channel && matches(typed.X)
	}
	return false
}

// Guaranteed requires a receive on every selected path.
// A mixed select does not settle every path. Its exact receive arm is credited
// on the edge, or at an unambiguous arm entry, keeping other cases open.
// https://github.com/containerd/stargz-snapshotter/blob/624678b4e421947534cbf0618f9609853cccee0f/store/manager.go#L193-L221
func Guaranteed(instruction ssa.Instruction, matches func(ssa.Value) bool) bool {
	if SelectedAtEntry(instruction, matches) {
		return true
	}
	if choice, ok := instruction.(*ssa.Select); ok {
		return choice.Blocking && len(choice.States) > 0 && !slices.ContainsFunc(choice.States, func(state *ssa.SelectState) bool {
			return state.Dir != types.RecvOnly || !matches(state.Chan)
		})
	}
	return From(instruction, matches)
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
