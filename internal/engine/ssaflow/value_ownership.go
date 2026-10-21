package ssaflow

import (
	"github.com/kojah/gohawk/internal/engine/ssaflow/valueforms"

	"golang.org/x/tools/go/ssa"
)

// Ownership provenance walks backward: given a value, which storage does it
// come out of, and does that storage outlive this call? It is the mirror of
// the forward question in store_escape.go, which asks where a value a function
// holds ends up. The two share a vocabulary and answer opposite questions, so
// they are kept apart: a proof that follows provenance must not accidentally
// pick up a rule about escape.

// ExternallyOwnedValue reports whether value comes from storage that outlives
// the current function invocation.
func ExternallyOwnedValue(value ssa.Value) bool {
	forms := TransparentChangeInterface | TransparentChangeType | TransparentConvert | TransparentMakeInterface
	return NewReachingWalk(forms).Any(value, externalOwnershipLeaf)
}

func externalOwnershipLeaf(walk ReachingWalk, value ssa.Value) bool {
	if source, ok := valueforms.OwnershipSource(value); ok {
		return walk.Any(source, externalOwnershipLeaf)
	}
	switch typed := value.(type) {
	case *ssa.Parameter, *ssa.FreeVar, *ssa.Global:
		return true
	case *ssa.Alloc:
		if typed.Referrers() != nil {
			for _, reference := range *typed.Referrers() {
				if store, ok := reference.(*ssa.Store); ok && store.Addr == typed && walk.Any(store.Val, externalOwnershipLeaf) {
					return true
				}
			}
		}
	}
	return false
}

// ElementOfAggregate reports whether value was selected from an element of
// a slice, array, map, or range iteration, directly or through fields and
// loads. Such a value is shared with whatever else holds the aggregate, so a
// lifecycle obligation attached to it cannot be attributed to the current
// function alone; callers treat it as unknown rather than violated.
func ElementOfAggregate(value ssa.Value) bool {
	forms := TransparentChangeInterface | TransparentChangeType | TransparentConvert | TransparentMakeInterface
	return NewReachingWalk(forms).Any(value, aggregateElementLeaf)
}

func aggregateElementLeaf(walk ReachingWalk, value ssa.Value) bool {
	switch typed := value.(type) {
	case *ssa.IndexAddr, *ssa.Index, *ssa.Lookup:
		return true
	case *ssa.Extract:
		if _, ok := typed.Tuple.(*ssa.Next); ok {
			return true
		}
	}
	source, ok := valueforms.OwnershipSource(value)
	return ok && walk.Any(source, aggregateElementLeaf)
}
