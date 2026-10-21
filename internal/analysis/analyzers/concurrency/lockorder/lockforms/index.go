package lockforms

import "golang.org/x/tools/go/ssa"

// IndexStep is a terminal answer or a single source for the existing reaching
// walk. Following a source here neither spends nor resets the walk allowance.
type IndexStep uint8

const (
	IndexStopped IndexStep = iota
	IndexDynamic
	IndexFollow
)

type indexSignature func(ssa.Value) (IndexStep, ssa.Value)

var _ indexSignature = DynamicIndex
