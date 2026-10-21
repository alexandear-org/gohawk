// Package httpacquire proves bounded HTTP acquisition exceptions. Exact HEAD
// provenance and local header-only servers share visible writer/default-effect
// searches. Missing, recursive or interrupted evidence cannot establish an
// ordinary body-bearing acquisition or a complete protocol exception.
package httpacquire

import proofs "github.com/kojah/gohawk/internal/engine/proof"

// Reason identifies the protocol evidence, independently of analyzer trace codes.
type Reason uint8

const (
	None Reason = iota
	Untouched
	BudgetExhausted
	HeadAcquisition
	HeadClientNotUnconfigured
	HeadRequestModified
	ClientOverrideUnresolved
	HandlerUnavailable
	HeaderOnlyEffects
	ServerIdentityUnavailable
	WriterEffectsUnavailable
)

// Proof retains an outcome and its protocol boundary. Unknown is never proof
// that closing is unnecessary or that ordinary resource flow may report.
type Proof struct {
	State  proofs.EvidenceState
	Reason Reason
}

func exhausted(budget *proofs.SearchBudget) bool { return budget.Exhausted() || budget.PoolExhausted() }

func (proof Proof) within(budget *proofs.SearchBudget) Proof {
	if exhausted(budget) {
		return Proof{State: proofs.EvidenceUnknown, Reason: BudgetExhausted}
	}
	return proof
}

func carriedProof(found bool, budget *proofs.SearchBudget) Proof {
	state := proofs.EvidenceDisproven
	if found {
		state = proofs.EvidenceProven
	}
	return (Proof{State: state, Reason: Untouched}).within(budget)
}

const (
	unknown   = proofs.EvidenceUnknown
	disproven = proofs.EvidenceDisproven
	proven    = proofs.EvidenceProven
)
