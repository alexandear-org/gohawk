package completionmapping

import (
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"golang.org/x/tools/go/ssa"
)

// These adapters expose storage/projection outcomes without applying mapping
// policy. In particular, stored containment retains its path even when the
// storage proof is unavailable: the preceding containment query admitted it.
func stable(context Context, value ssa.Value, invocation ssa.Instruction) heapmodel.StoredValue {
	return heapmodel.NewStorage(context.Budget).StableContent(value, invocation)
}

func same(context Context, argument, target ssa.Value) bool {
	return heapmodel.NewStorage(context.Budget).Same(argument, target).Proven()
}

func ambiguous(argument, target ssa.Value) bool {
	return heapmodel.MayAlias(argument, target) && !heapmodel.DefinitelySameValue(argument, target)
}

func identity(context Context, argument, target ssa.Value) bool {
	return ssaflow.ProveIdentityWithin(ssaflow.AccessPath{Value: argument}, ssaflow.AccessPath{Value: target}, context.Budget).Proven()
}

func directCell(context Context, binding ssa.Value) (*ssa.Alloc, bool) {
	cell, ok := binding.(*ssa.Alloc)
	return cell, ok && context.Queries.DirectStore(cell)
}

func capturedOwner(context Context, local, binding ssa.Value, invocation ssa.Instruction) (Local, bool) {
	cell, ok := binding.(*ssa.Alloc)
	if !ok {
		return Local{}, false
	}
	stored := heapmodel.NewStorage(context.Budget).StableContent(cell, invocation)
	return Local{Local: local, Supplied: binding, Kind: Owner}, stored.Proven()
}

func unmatchedContainment(context Context, binding, target ssa.Value, matched bool) bool {
	return !matched && context.Queries.MayContain(binding, target)
}

type pathState uint8

const (
	pathMissing pathState = iota
	pathFound
	pathCutoff
)

type pathEvidence struct {
	State pathState
	Path  []string
}

func projection(context Context, argument, target ssa.Value) pathEvidence {
	proof := heapmodel.ProveStrictProjectionPathWithin(argument, target, context.Budget)
	return pathResult(proof.Path, proof.Reason, proof.Proven())
}

func storedPath(context Context, argument, target ssa.Value, invocation ssa.Instruction) pathEvidence {
	proof := heapmodel.ProveStoredPathWithin(argument, target, invocation, context.Budget)
	return pathResult(proof.Path, proof.Reason, true)
}

func pathResult(path []string, reason proofs.EvidenceReason, found bool) pathEvidence {
	if reason == proofs.EvidenceBudgetExhausted {
		return pathEvidence{State: pathCutoff}
	}
	if found {
		return pathEvidence{State: pathFound, Path: path}
	}
	return pathEvidence{State: pathMissing}
}

func owner(context Context, argument, target ssa.Value) (ssa.Value, bool) {
	value := StorageOwner(target, argument, context.Budget)
	return value, value != nil
}
