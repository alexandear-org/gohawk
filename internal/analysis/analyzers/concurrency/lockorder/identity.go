package lockorder

import (
	"github.com/kojah/gohawk/internal/analysis/analyzers/concurrency/lockorder/lockidentity"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"golang.org/x/tools/go/ssa"
)

// lockIdentityOf names the lock a receiver value denotes, or returns the
// empty string when its origin cannot be told apart from another lock's.
func lockIdentityOf(value ssa.Value) string { return lockIdentityWithin(value, nil) }

// lockIdentityWithin keeps storage, reaching and cutoff policy at one boundary.
func lockIdentityWithin(value ssa.Value, budget *proofs.SearchBudget) string {
	return lockidentity.Within(value, budget)
}

// privateMutexOnly reports the deliberately narrow case where a local mutex
// allocation is used exclusively by direct synchronous mutex operations.
// No pointer, callback, or owner leaves these uses, so a held mutex cannot
// affect another caller after return. This says nothing about recursive
// acquisition before return, which is still checked.
// https://github.com/alajmo/sake/blob/86986df901293db0f7d1e548ef34c849bb1f709d/core/run/exec.go#L1070-L1084
func privateMutexOnly(value ssa.Value, budget *proofs.SearchBudget) bool {
	allocation, ok := value.(*ssa.Alloc)
	if !ok || allocation.Referrers() == nil {
		return false
	}
	for _, use := range *allocation.Referrers() {
		if !budget.Spend() {
			return false
		}
		if _, ok := use.(*ssa.DebugRef); ok {
			continue
		}
		if _, ok := use.(*ssa.Call); !ok {
			return false
		}
		_, _, receiver, ok := mutexActionWithin(use, budget)
		if !ok || receiver != allocation {
			return false
		}
	}
	return true
}
