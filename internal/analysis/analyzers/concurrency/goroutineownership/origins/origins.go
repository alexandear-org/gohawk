// Package origins distinguishes possible factory or registry ownership from
// a caller's fresh completion handle. Each query shares its allowance across
// reaching values and storage; incomplete evidence remains unknown.
package origins

import (
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"golang.org/x/tools/go/ssa"
)

// Origin queries name their own transparent forms so containment changes
// cannot silently widen factory provenance.
const originForms = ssaflow.TransparentChangeInterface | ssaflow.TransparentChangeType | ssaflow.TransparentConvert | ssaflow.TransparentMakeInterface

func factoryOriginProof(found bool, budget *proofs.SearchBudget) proofs.Proof {
	if budget.Exhausted() {
		return proofs.Proof{State: proofs.EvidenceUnknown, Reason: proofs.EvidenceBudgetExhausted}
	}
	if found {
		return proofs.Proof{State: proofs.EvidenceProven, Reason: proofs.EvidenceStructuralWalk, Provenance: proofs.EvidenceFromLocalSSA}
	}
	return proofs.Proof{State: proofs.EvidenceDisproven, Reason: proofs.EvidenceNotFound}
}

// Signal finds possible factory ownership of a completion channel.
// A channel supplied by a factory or registry may already have another owner.
// Until its allocation and ownership are established, a local launch cannot
// manufacture an exclusive receive obligation for this caller. This is not a
// claim that an arbitrary factory channel is drained.
// https://github.com/deckarep/golang-set/blob/711c30df0fdf98710a4ca0211e12ef7210967ad3/threadsafe.go#L268-L287
func Signal(value ssa.Value, spawn *ssa.Go, budget *proofs.SearchBudget) proofs.Proof {
	storage := heapmodel.NewStorage(budget)
	var leaf func(ssaflow.ReachingWalk, ssa.Value) bool
	leaf = func(walk ssaflow.ReachingWalk, current ssa.Value) bool {
		if resolved := storage.Resolve(current); resolved.Proven() && resolved.Value != current {
			return walk.Any(resolved.Value, leaf)
		}
		// A returned struct is copied into a local before invoking its pointer
		// method. That copy does not manufacture exclusive ownership of the
		// factory's channel; a companion result may be its actual join handle.
		// https://github.com/tus/tusd/blob/c9d174d0e20c69f24e9785d2f639df4da1c4fdc5/pkg/s3store/s3store_part_producer_test.go#L28-L56
		if _, allocated := current.(*ssa.Alloc); allocated {
			if content := storage.Content(current, spawn); content.Proven() && content.Value != current {
				return walk.Any(content.Value, leaf)
			}
		}
		if result, ok := current.(*ssa.Extract); ok {
			return walk.Any(result.Tuple, leaf)
		}
		_, call := current.(*ssa.Call)
		return call
	}
	return factoryOriginProof(ssaflow.NewReachingWalk(originForms).Within(budget).Any(value, leaf), budget)
}

// Group finds possible registry ownership of a settling group.
// An opaque producer can lend a registry-owned group, not allocate a new one.
// Without its body we cannot assign the join obligation to this invocation.
// This is uncertainty, not proof that the caller or registry actually waits.
// https://github.com/i-love-flamingo/flamingo/blob/79a55d62bb7a1bffe11a4dea1444490b14785879/core/requesttask/filter.go#L28-L60
func Group(value ssa.Value, budget *proofs.SearchBudget) proofs.Proof {
	storage := heapmodel.NewStorage(budget)
	var leaf func(ssaflow.ReachingWalk, ssa.Value) bool
	leaf = func(walk ssaflow.ReachingWalk, current ssa.Value) bool {
		if resolved := storage.Resolve(current); resolved.Proven() && resolved.Value != current {
			return walk.Any(resolved.Value, leaf)
		}
		switch typed := current.(type) {
		case *ssa.FieldAddr:
			// An embedded group retains its aggregate's possible registry owner.
			// Resolve loads first: a fresh pointer stored into an owner's group
			// field must not borrow the owner's opaque factory provenance.
			// https://github.com/FDio/govpp/blob/c71484d8c74da940abbd70407b53894fa4c56f01/extras/gomemif/examples/bridge/bridge.go#L50-L100
			return walk.Any(typed.X, leaf)
		case *ssa.Extract:
			return walk.Any(typed.Tuple, leaf)
		case *ssa.Call:
			callee, _ := ssacall.DirectCallee(typed.Common())
			return callee == nil || len(callee.Blocks) == 0
		}
		return false
	}
	return factoryOriginProof(ssaflow.NewReachingWalk(originForms|ssaflow.TransparentTypeAssert).Within(budget).Any(value, leaf), budget)
}
