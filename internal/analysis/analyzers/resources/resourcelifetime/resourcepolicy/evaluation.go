package resourcepolicy

import (
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"golang.org/x/tools/go/ssa"
)

// Decision says whether evidence permits a diagnostic. Disproven accepts the
// candidate without proving cleanup; unknown suppresses an incomplete proof.
// Leak is the exact return witness supplied by a reportable flow.
type Decision struct {
	State  proofs.EvidenceState
	Reason Reason
	Leak   *ssa.Return
}

// EvaluationQueries delays candidate effects until the policy exclusion fails.
type EvaluationQueries interface {
	MemoryWriter() bool
	Begin() AcquisitionQueries
}

// AcquisitionQueries keeps pre-flow evidence on one candidate allowance.
type AcquisitionQueries interface {
	Error() PairedError
	Boundary() Reason
	Canceled() Evidence
	Asserted(ssa.Value) Evidence
	Optional(ssa.Value) Correlation
}

// Initialization binds one complete correlation before collecting flow inputs.
type Initialization interface{ Prepare() Prepared }

// FlowQueries evaluates the flow once and supplies only reached final exclusions.
type FlowQueries interface {
	Flow(ssa.Value) Decision
	DatabaseCleanup() bool
	PoolExhausted() bool
	Exit() ExitEvidence
}

// PairedError retains the exact error value even for a completed negative proof.
type PairedError struct {
	State  proofs.EvidenceState
	Reason Reason
	Value  ssa.Value
}

// Correlation carries the initialization context belonging to this answer.
// An unknown answer must never initialize or select its resource phi.
type Correlation struct {
	State      proofs.EvidenceState
	Reason     Reason
	Initialize Initialization
}

// Prepared carries the flow context constructed from a complete correlation.
// Its preparation evidence must be available before the flow is queried.
type Prepared struct {
	State   proofs.EvidenceState
	Reason  Reason
	Queries FlowQueries
}

// ExitEvidence retains the outcome and availability of the same child query.
// Positive process-exit evidence precedes its exhausted-child observation.
type ExitEvidence struct{ Reclaimed, Exhausted bool }

// Unknown reports that error-value lookup cannot activate the obligation.
func (p PairedError) Unknown() bool { return p.State == proofs.EvidenceUnknown }

// Unknown prevents incomplete correlation from reaching initialization.
func (p Correlation) Unknown() bool { return p.State == proofs.EvidenceUnknown }

// Unknown prevents incomplete setup from reaching the flow.
func (p Prepared) Unknown() bool { return p.State == proofs.EvidenceUnknown }

// Reportable distinguishes a diagnostic witness from accepted or opaque flow.
func (p Decision) Reportable() bool { return p.State == proofs.EvidenceProven }

func noBoundary(reason Reason) bool       { return reason == None }
func exitReclaimed(e ExitEvidence) bool   { return e.Reclaimed }
func exitInterrupted(e ExitEvidence) bool { return e.Exhausted }

type (
	evaluationSignature  func(EvaluationQueries) Decision
	acquisitionSignature func(AcquisitionQueries) Decision
	finalFlowSignature   func(FlowQueries, ssa.Value) Decision
)

var (
	_ evaluationSignature  = Evaluate
	_ acquisitionSignature = Acquire
	_ finalFlowSignature   = Finalize
)

// PreparationQueries collects classifier inputs in order. Guarded first binds
// the local collection, then discovers guarded defers. Prior commits any
// positive trace observation before returning its evidence.
type PreparationQueries interface {
	Deferred() Evidence
	Owners() bool
	Guarded() Evidence
	Prior() Evidence
}
type preparationSignature func(PreparationQueries) Evidence

var _ preparationSignature = PrepareFlow
