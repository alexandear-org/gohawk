// Package lockvariant identifies receivers selected by a loop iteration.
// This is uncertainty, not freshness: even a repeated receive may return the
// same pointer. Caller-owned reaching state bounds recursive source queries.
package lockvariant

import (
	"go/token"
	"slices"

	"github.com/kojah/gohawk/internal/engine/ssaflow"
	cfg "github.com/kojah/gohawk/internal/engine/ssaflow/cfg"
	"golang.org/x/tools/go/ssa"
)

// Value preserves the caller's walk and tests each SSA value at most once.
// Fixed receiver fields and globals do not become variant just by being locked
// in a loop; the selected value must derive from an iteration-dependent source.
func Value(walk ssaflow.ReachingWalk, value ssa.Value) bool {
	if value == nil || !walk.Mark(value) {
		return false
	}
	return decideValue(walk, value)
}

type (
	valueSignature   func(ssaflow.ReachingWalk, ssa.Value) bool
	extractSignature func(ssaflow.ReachingWalk, *ssa.Extract) bool
)

var (
	_ valueSignature   = decideValue
	_ extractSignature = decideExtract
)

const receiveToken = token.ARROW

func cycle(instruction ssa.Instruction) bool { return cfg.BlockInCycle(instruction.Block()) }

func arguments(walk ssaflow.ReachingWalk, call *ssa.Call) bool {
	return slices.ContainsFunc(call.Common().Args, func(argument ssa.Value) bool {
		return Value(walk, argument)
	})
}

func transparent(value ssa.Value) (ssa.Value, bool) {
	return ssaflow.UnwrapTransparentValue(value,
		ssaflow.TransparentChangeInterface|ssaflow.TransparentChangeType|ssaflow.TransparentConvert|ssaflow.TransparentMakeInterface)
}
