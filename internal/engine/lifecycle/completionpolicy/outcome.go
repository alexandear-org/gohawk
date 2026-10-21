// Package completionpolicy selects the outcome of a completed or abandoned
// completion search. It does not traverse SSA or spend the search allowance.
package completionpolicy

import proofs "github.com/kojah/gohawk/internal/engine/proof"

type signature func(*proofs.SearchBudget, bool, bool, bool) proofs.Proof

var _ signature = decideUnproven

// Unproven distinguishes missing completion from a search that cannot decide.
// searched records body availability, including when the search later stops.
func Unproven(budget *proofs.SearchBudget, searched, incomplete, inCycle bool) proofs.Proof {
	return decideUnproven(budget, searched, incomplete, inCycle)
}

func exhausted(budget *proofs.SearchBudget) bool { return budget.Exhausted() }

const (
	unknown           = proofs.EvidenceUnknown
	disproven         = proofs.EvidenceDisproven
	unavailable       = proofs.EvidenceUnavailable
	budgetExhausted   = proofs.EvidenceBudgetExhausted
	completionInCycle = proofs.EvidenceCompletionInCycle
	notFound          = proofs.EvidenceNotFound
	localSSA          = proofs.EvidenceFromLocalSSA
	noProvenance      = proofs.EvidenceProvenance(0)
)

func evidence(state proofs.EvidenceState, reason proofs.EvidenceReason, provenance proofs.EvidenceProvenance) proofs.Proof {
	return proofs.Proof{State: state, Reason: reason, Provenance: provenance}
}
