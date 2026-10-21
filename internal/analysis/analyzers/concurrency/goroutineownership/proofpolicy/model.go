// Package proofpolicy selects the final goroutine ownership proof from ordered
// evidence. The consuming analyzer owns searches, flow evaluation, allowances
// and trace effects; an answer, including unknown, stops further alternatives.
package proofpolicy

import (
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	ssapath "github.com/kojah/gohawk/internal/engine/ssaflow/path"
	"golang.org/x/tools/go/ssa"
)

// Outcome distinguishes proven lifecycle behavior from an opaque handoff.
type Outcome uint8

const (
	Unknown Outcome = iota
	Honored
	Violated
	Transferred
)

// Proof is the authoritative outcome consumed by reporting and other analyzers.
// Witness is the return reached without a join when the obligation is violated.
type Proof struct {
	Outcome Outcome
	Reason  Reason
	Witness *ssa.Return
}

// Discovery retains the availability and obligation facts before any search.
type Discovery struct{ Exhausted, Obligation, EarlyDone bool }

// Flow retains one obligation walk's outcome and violating return witness.
type Flow struct {
	Outcome ssapath.ObligationOutcome
	Witness *ssa.Return
}

// RelayKind distinguishes related, absent, and interrupted dependency evidence.
type RelayKind uint8

const (
	RelayAbsent RelayKind = iota
	RelayRelated
	RelayInterrupted
)

// Relay retains the interrupted allowance for the original cutoff observation.
type Relay struct {
	Kind   RelayKind
	Budget *proofs.SearchBudget
}

// CensusKind separates a completed signal census from unavailable evidence.
type CensusKind uint8

const (
	CensusObserved CensusKind = iota
	CensusUnobserved
	CensusUnavailable
	CensusInterrupted
)

// Census retains a child cutoff independently of the shared pool's availability.
type Census struct {
	Kind   CensusKind
	Budget *proofs.SearchBudget
}

// ProofQueries executes reached top-level evidence and records negative
// considered reasons immediately, before the next query. Flow is evaluated
// once; Honored and Uncertain observe that same value.
type ProofQueries interface {
	Lifecycle() (Proof, bool)
	Dominating() (Proof, bool)
	Workers() bool
	Flow() Flow
	Honored(Flow) bool
	Guarded() bool
	Uncertain(Flow) bool
	Fallback(*ssa.Return) Proof
}

// LifecycleQueries preserves the grouped lifetime evidence and its allowances.
type LifecycleQueries interface {
	Relay() Relay
	RelayCutoff(*proofs.SearchBudget) Proof
	CompletionHandle() (Proof, bool)
	CallerLifetime() (Proof, bool)
	Synctest() bool
}

// FallbackQueries executes the remaining precision checks after local flow.
type FallbackQueries interface {
	FlagGuarded() bool
	Counted() bool
	SharedStorage() bool
	Buffered() bool
	Census() Census
	CensusCutoff(*proofs.SearchBudget) Proof
	ProgramEntry() bool
}

type (
	proofSignature     func(Discovery, ProofQueries) Proof
	fallbackSignature  func(FallbackQueries, *ssa.Return) Proof
	lifecycleSignature func(LifecycleQueries) (Proof, bool)
)

var (
	_ proofSignature     = Prove
	_ fallbackSignature  = Fallback
	_ lifecycleSignature = Lifecycle
)

func emptyProof() Proof { return Proof{} }
