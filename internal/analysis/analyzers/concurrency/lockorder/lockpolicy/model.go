// Package lockpolicy selects lock diagnostic evidence after the analyzer's
// bounded searches. Unknown never establishes release or field protection.
// Candidate iteration and state effects stay with the consuming flow.
package lockpolicy

import proofs "github.com/kojah/gohawk/internal/engine/proof"

// Proof is the authoritative diagnostic disposition and its stable reason.
type Proof struct {
	State  proofs.EvidenceState
	Reason Reason
}

// Step distinguishes a terminal decision from evidence retained while the next
// matching owner is inspected. Ordinary uncertainty continues; cutoff stops.
type Step struct {
	Stop  bool
	Proof Proof
}

// Answer retains the contract's exact reason and its post-query availability.
// Exhausted evidence takes precedence over a positive witness.
type Answer struct {
	Proven    bool
	Reason    Reason
	Exhausted bool
}

// MissingQueries supplies evidence for one held-return lock identity.
type MissingQueries interface {
	Spend() bool
	Uncertain() bool
	Private() Answer
	Released() bool
	Held() Answer
	Caller() Answer
	Returns() bool
}

// ReadWriteQueries supplies evidence for an already matched write/lock owner.
type ReadWriteQueries interface {
	Guard() Proof
	PrivateStorage() bool
	HeldWriter() bool
	Writer() WriterEvidence
}

// WriterEvidence retains the shared writer search's availability contract.
type WriterEvidence = proofs.Proof

// Unknown identifies an opaque field contract without treating it as safety.
func (p Proof) Unknown() bool { return p.State == proofs.EvidenceUnknown }

const (
	unknownState   = proofs.EvidenceUnknown
	disprovenState = proofs.EvidenceDisproven
	provenState    = proofs.EvidenceProven
)

type (
	missingSignature   func(MissingQueries) Proof
	readWriteSignature func(ReadWriteQueries) Step
)

var (
	_ missingSignature   = MissingRelease
	_ readWriteSignature = ReadWrite
)
