// Package completionmapping chooses how a caller's target is represented by
// a callee local. Rules sequence evidence; the consuming search owns recursion,
// callback/containment queries, memo invalidation and the shared allowance.
package completionmapping

import (
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"golang.org/x/tools/go/ssa"
)

// Kind describes the supplied value's relation to the caller's target.
type Kind uint8

const (
	// Exact names the target or an aggregate that contains it.
	Exact Kind = iota
	// Projection names a stable proper projection; only that local can settle it.
	Projection
	// Owner requires the receiver to select the mirrored path beneath the local.
	Owner
	// Callback names a callback bound to the target's completion method.
	Callback
)

// Local maps a callee parameter/capture to its caller-supplied value. Path is
// the target's path beneath an Exact aggregate, or a Projection beneath the
// target. A nonempty path never permits cleanup of a sibling field.
type Local struct {
	Local    ssa.Value
	Supplied ssa.Value
	Kind     Kind
	Path     []string
}

// Queries supplies evidence owned by the enclosing completion search. Calls
// execute only when reached; failed strategies never refund work or memo effects.
type Queries interface {
	Callback(ssa.Value, ssa.Value) bool
	MayContain(ssa.Value, ssa.Value) bool
	DirectStore(ssa.Value) bool
	DeferredValue(ssa.Value, ssa.Value, ssa.Instruction) heapmodel.StoredValue
	StoredOnPath(ssa.Value, ssa.Value, ssa.Instruction) bool
	// Uncertain marks a possible alias without invalidating the memo.
	Uncertain() (Local, bool)
	// Cutoff also invalidates the enclosing memo answer before returning failure.
	Cutoff() (Local, bool)
}

// Context keeps request-local evidence and flags explicit. Deferred is true
// only for a deferred invocation with a known observation instruction.
type Context struct {
	Queries  Queries
	Budget   *proofs.SearchBudget
	Exact    bool
	Deferred bool
}

type (
	pathSignature         func(Local, ssa.Value, pathList, bool, *proofs.SearchBudget) (string, bool)
	captureSignature      func(Context, ssa.Value, ssa.Value, ssa.Value, ssa.Instruction) (Local, bool)
	relationSignature     func(Context, ssa.Value, ssa.Value, ssa.Value, ssa.Value, bool, bool, ssa.Instruction) (Local, bool)
	argumentSignature     func(Context, ssa.Value, ssa.Value, ssa.Value, ssa.Instruction) (Local, bool)
	deferredSignature     func(Context, ssa.Value, *ssa.Alloc, ssa.Value, ssa.Instruction) (Local, bool)
	relationCellSignature func(Context, ssa.Value, *ssa.Alloc, ssa.Value, ssa.Instruction, heapmodel.DeferredCellMatch) (Local, bool)
	receiverSignature     func(Local, ssa.Value, ssa.Value, *proofs.SearchBudget) ReceiverProof
	pathList              = []string
	storedValue           = heapmodel.StoredValue
)

var (
	_ pathSignature                                   = TranslatePath
	_ func(ssa.Value, *proofs.SearchBudget) ssa.Value = ssaflow.CapturedBindingValueWithin
	_ captureSignature                                = Captured
	_ relationSignature                               = CaptureRelation
	_ argumentSignature                               = Argument
	_ deferredSignature                               = DeferredCell
	_ relationCellSignature                           = RelateCell
	_ receiverSignature                               = Receives
)

func emptyLocal() Local                    { return Local{} }
func identical(left, right ssa.Value) bool { return left == right }
func deferred(context Context) bool        { return context.Deferred }
func exhausted(context Context) bool {
	return context.Budget.Exhausted() || context.Budget.PoolExhausted()
}

const (
	deferredExact    = heapmodel.DeferredCellExact
	deferredContains = heapmodel.DeferredCellContains
	deferredUnknown  = heapmodel.DeferredCellUnknown
)
