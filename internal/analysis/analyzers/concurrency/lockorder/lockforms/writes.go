// Package lockforms selects lock-related SSA forms and their exact operands.
// It performs no alias, ownership or control-flow search. Consumers apply their
// own identity and uncertainty policy to each recognized step.
package lockforms

import (
	"go/types"

	"github.com/kojah/gohawk/internal/engine/ssaflow"

	"golang.org/x/tools/go/ssa"
)

type (
	destinationSignature func(ssa.Instruction) (ssa.Value, bool)
	ownerStepSignature   func(ssa.Value) (ssa.Value, bool)
)

var (
	_ destinationSignature = Destination
	_ ownerStepSignature   = OwnerStep
)

// builtinArguments exposes a builtin's name only after establishing declaration
// identity through its SSA builtin form. Project-defined functions cannot match.
func builtinArguments(common ssa.CallCommon) (string, ssa.Value, bool) {
	builtin, ok := common.Value.(*ssa.Builtin)
	if !ok || len(common.Args) == 0 {
		return "", nil, false
	}
	return builtin.Name(), common.Args[0], true
}

func indexedSource(source ssa.Value) ssa.Value { //nolint:ireturn // Keep original storage identity.
	// Only a slice load retains the owner's backing array. A loaded pointer
	// names another object and must remain opaque to this address walk.
	if _, slice := source.Type().Underlying().(*types.Slice); slice {
		if address, ok := ssaflow.IdentitySource(source); ok {
			return address
		}
	}
	return source
}
