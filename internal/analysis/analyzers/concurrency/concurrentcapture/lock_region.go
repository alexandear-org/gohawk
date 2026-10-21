package concurrentcapture

import (
	"github.com/kojah/gohawk/internal/analysis/passes/concurrencyfacts"
	"golang.org/x/tools/go/ssa"
)

// lockRegion folds a complete ordered prefix of synchronization effects.
// It answers whether any exact mutex is held at a point; callers decide what
// that evidence means for their own diagnostic. Indirect locks, unmatched
// unlocks, and condition waits make the result unknown.
type lockRegion struct {
	held    map[ssa.Value]bool
	unknown bool
}

// apply consumes effects in execution order. It never infers a missing effect
// from an empty sequence: the caller must have established prefix completeness.
func (region *lockRegion) apply(operations []concurrencyfacts.Operation) {
	for _, operation := range operations {
		effect, value := lockTransition(operation, region)
		switch effect {
		case lockEffectUnknown:
			region.unknown = true
		case lockEffectAcquire:
			if region.held == nil {
				region.held = make(map[ssa.Value]bool)
			}
			region.held[value] = true
		case lockEffectRelease:
			delete(region.held, value)
		case lockEffectIgnore:
		}
	}
}

// heldState reports whether an exact mutex is held and whether the prefix was
// sufficiently modeled to decide that. Unknown is not evidence of no lock.
func (region *lockRegion) heldState() (held bool, known bool) {
	return len(region.held) != 0, !region.unknown
}

// A transition is selected before the caller changes the held set. Uncertainty
// remains sticky in lockRegion even when later exact operations can be applied.
type lockEffect uint8

const (
	lockEffectIgnore lockEffect = iota
	lockEffectUnknown
	lockEffectAcquire
	lockEffectRelease
)

type lockTransitionSignature func(concurrencyfacts.Operation, *lockRegion) (lockEffect, ssa.Value)

var _ lockTransitionSignature = lockTransition

const (
	lock       = concurrencyfacts.Lock
	unlock     = concurrencyfacts.Unlock
	readLock   = concurrencyfacts.ReadLock
	readUnlock = concurrencyfacts.ReadUnlock
	condWait   = concurrencyfacts.CondWait
)

func projected(operation concurrencyfacts.Operation) bool {
	return operation.Resource.Projection.Depth != 0
}

func (region *lockRegion) holds(value ssa.Value) bool { return region.held[value] }
