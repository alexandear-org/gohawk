// Package containment identifies possible possession of tracked worker values.
// Old stores, mixed choices and aggregate projections remain possible evidence;
// they do not establish exact identity or an ownership transfer.
package containment

import (
	"slices"

	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"golang.org/x/tools/go/ssa"
)

// forms are the wrappers transparent to captured-binding containment.
const forms = ssaflow.TransparentChangeInterface | ssaflow.TransparentChangeType | ssaflow.TransparentConvert | ssaflow.TransparentMakeInterface

// Matches checks possible containment through the caller's reaching walk.
func Matches(walk ssaflow.ReachingWalk, value, target ssa.Value) bool {
	if value != nil && heapmodel.MayAlias(value, target) {
		return true
	}
	return walk.Any(value, func(walk ssaflow.ReachingWalk, value ssa.Value) bool {
		if heapmodel.MayAlias(value, target) {
			return true
		}
		return decideContainment(walk, value, target)
	})
}

// callResultCarries assumes a non-builtin call retains its arguments, because
// wrapSignal(done) may return the owner that later escapes. Builtins other
// than append only observe their arguments.
func callResultCarries(walk ssaflow.ReachingWalk, call *ssa.Call, target ssa.Value) bool {
	if builtin, ok := call.Common().Value.(*ssa.Builtin); ok && builtin.Name() != "append" {
		return false
	}
	return slices.ContainsFunc(call.Common().Args, func(argument ssa.Value) bool {
		return Matches(walk, argument, target)
	})
}

// storedCarries follows stores into an addressable local and into the fields
// and elements selected from it. Every `results[i]` expression is its own
// IndexAddr, and every `w.field` expression its own FieldAddr, so a read
// through one is matched against stores made through any other address of the
// same element or field. This over-approximation does not establish current
// field contents or exact value identity.
func storedCarries(walk ssaflow.ReachingWalk, address, target ssa.Value) bool {
	if siblingSelectionCarries(walk, address, target) {
		return true
	}
	if address.Referrers() == nil {
		return false
	}
	for _, reference := range *address.Referrers() {
		if decideStoredUse(walk, reference, address, target) {
			return true
		}
	}
	return false
}

// siblingSelectionCarries matches a read address against stores made through
// any other address selecting the same element or field of the same aggregate.
// A composite literal stores through one field address while a later read
// selects that field through another, so the store is only visible from the
// sibling.
func siblingSelectionCarries(walk ssaflow.ReachingWalk, address, target ssa.Value) bool {
	base, selectsSame := siblingSelection(address)
	if base == nil || base.Referrers() == nil {
		return false
	}
	for _, sibling := range *base.Referrers() {
		other, ok := sibling.(ssa.Value)
		if !ok || other == address || !selectsSame(sibling) || !walk.Mark(other) {
			continue
		}
		if storedCarries(walk, other, target) {
			return true
		}
	}
	return false
}

// siblingSelection returns the aggregate an address selects from and a
// predicate matching other addresses that select the same element or field.
func siblingSelection(address ssa.Value) (ssa.Value, func(ssa.Instruction) bool) {
	switch typed := address.(type) {
	case *ssa.IndexAddr:
		return typed.X, func(instruction ssa.Instruction) bool {
			_, ok := instruction.(*ssa.IndexAddr)
			return ok
		}
	case *ssa.FieldAddr:
		return typed.X, func(instruction ssa.Instruction) bool {
			other, ok := instruction.(*ssa.FieldAddr)
			return ok && other.Field == typed.Field
		}
	}
	return nil, nil
}

// Binding matches a closure binding or call argument against a tracked
// value, including an addressable local that has contained it.
func Binding(binding, target ssa.Value) bool {
	return heapmodel.CapturedBindingMatches(binding, target) || Matches(ssaflow.NewReachingWalk(forms), binding, target)
}
