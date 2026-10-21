package completionmapping

import (
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"golang.org/x/tools/go/ssa"
)

// ReceiverProof distinguishes a possible aggregate cleanup from an exact
// receiver match. Only an exact match can establish must-complete coverage.
type ReceiverProof struct {
	proofs.Proof
	Possible bool
}

// MatchProof translates a mechanical match without promoting uncertainty.
func MatchProof(matches bool) ReceiverProof {
	state, reason := proofs.EvidenceDisproven, proofs.EvidenceUnavailable
	if matches {
		state, reason = proofs.EvidenceProven, proofs.EvidenceCalledCompletion
	}
	return ReceiverProof{Proof: proofs.Proof{State: state, Reason: reason}}
}

func possible() ReceiverProof {
	return ReceiverProof{Proof: proofs.Proof{State: proofs.EvidenceUnknown, Reason: proofs.EvidenceUnavailable}, Possible: true}
}

func mirrored(local Local, receiver, target ssa.Value, budget *proofs.SearchBudget) ReceiverProof {
	return ReceiverProof{Proof: ssaflow.ProveIdentityWithin(ssaflow.AccessPath{Value: receiver, Root: local.Local},
		ssaflow.AccessPath{Value: target, Root: local.Supplied}, budget).Proof}
}

func relativePath(receiver, local ssa.Value, budget *proofs.SearchBudget) (pathList, bool) {
	return heapmodel.AccessPathFromParameterWithin(receiver, local, budget)
}

func nonempty(path []string) bool { return len(path) > 0 }
func samePath(actual, expected []string) bool {
	return ssaflow.JoinAccessPath(actual) == ssaflow.JoinAccessPath(expected)
}
