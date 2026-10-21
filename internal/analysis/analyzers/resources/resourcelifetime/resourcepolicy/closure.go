package resourcepolicy

import "golang.org/x/tools/go/ssa"

// ClosureQueries supplies evidence at the closure's invocation or deferred
// observation point. Guarded-body and ordinary capture queries retain separate
// allowances; a known capture does not excuse unavailable cleanup evidence.
type ClosureQueries interface {
	Deferred(*ssa.Defer) Evidence
	Guarded(ssa.Instruction, *ssa.MakeClosure) Evidence
	Owner(*ssa.MakeClosure) Evidence
	Capture(*ssa.MakeClosure) Evidence
	Unreadable(*ssa.MakeClosure) bool
	Retains(*ssa.MakeClosure) bool
}

type closureSignature func(ClosureQueries, ssa.Instruction, *ssa.MakeClosure, bool) (Reason, bool)

var (
	_ closureSignature = Closure
	_ closureSignature = ClosureBody
	_ closureSignature = ClosureCapture
)

type closureFinishSignature func(ClosureQueries, ssa.Instruction, *ssa.MakeClosure) (Reason, bool)

var _ closureFinishSignature = ClosureFinish
