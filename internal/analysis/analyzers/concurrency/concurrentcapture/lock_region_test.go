package concurrentcapture

import (
	"testing"

	"github.com/kojah/gohawk/internal/analysis/passes/concurrencyfacts"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"golang.org/x/tools/go/ssa"
)

func TestLockRegionTracksExactHeldMutexes(t *testing.T) {
	mutex := new(ssa.Alloc)
	other := new(ssa.Alloc)
	operation := func(kind concurrencyfacts.Kind, resource ssa.Value) concurrencyfacts.Operation {
		return concurrencyfacts.Operation{Kind: kind, Resource: concurrencyfacts.Reference{Value: resource}}
	}
	var region lockRegion
	region.apply([]concurrencyfacts.Operation{operation(concurrencyfacts.Lock, mutex), operation(concurrencyfacts.Lock, other)})
	if held, known := region.heldState(); !held || !known {
		t.Fatalf("two held locks = (%v, %v)", held, known)
	}
	region.apply([]concurrencyfacts.Operation{operation(concurrencyfacts.Unlock, mutex)})
	if held, known := region.heldState(); !held || !known {
		t.Fatalf("one held lock = (%v, %v)", held, known)
	}
	region.apply([]concurrencyfacts.Operation{operation(concurrencyfacts.Unlock, other)})
	if held, known := region.heldState(); held || !known {
		t.Fatalf("released locks = (%v, %v)", held, known)
	}
}

func TestLockRegionDeclinesUncertainRelease(t *testing.T) {
	for _, operation := range []concurrencyfacts.Operation{
		{Kind: concurrencyfacts.Unlock, Resource: concurrencyfacts.Reference{Value: new(ssa.Alloc)}},
		{Kind: concurrencyfacts.Lock, Resource: concurrencyfacts.Reference{Value: new(ssa.Alloc), Indirect: true}},
		{Kind: concurrencyfacts.CondWait},
		{Kind: concurrencyfacts.ReadLock},
		{Kind: concurrencyfacts.ReadUnlock},
		{Kind: concurrencyfacts.Lock},
		{Kind: concurrencyfacts.Unlock},
		{Kind: concurrencyfacts.Send, Resource: concurrencyfacts.Reference{Projection: ssaflow.EmbeddedFieldPath{Depth: 1}}},
	} {
		var region lockRegion
		region.apply([]concurrencyfacts.Operation{operation})
		if _, known := region.heldState(); known {
			t.Errorf("uncertain operation %+v produced a known region", operation)
		}
	}
}

// Later exact effects must not erase earlier uncertainty. They still update
// the exact held set so an uncertain prefix does not invent a held mutex.
func TestLockRegionUncertaintySurvivesExactEffects(t *testing.T) {
	mutex := new(ssa.Alloc)
	var region lockRegion
	region.apply([]concurrencyfacts.Operation{
		{Kind: concurrencyfacts.CondWait},
		{Kind: concurrencyfacts.Lock, Resource: concurrencyfacts.Reference{Value: mutex}},
		{Kind: concurrencyfacts.Send},
		{Kind: concurrencyfacts.Unlock, Resource: concurrencyfacts.Reference{Value: mutex}},
	})
	if held, known := region.heldState(); held || known {
		t.Fatalf("uncertain prefix with settled mutex = (%v, %v), want (false, false)", held, known)
	}
}
