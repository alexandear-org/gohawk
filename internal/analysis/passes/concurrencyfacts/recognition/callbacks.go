package recognition

import (
	"go/types"

	"github.com/kojah/gohawk/internal/analysis/passes/concurrencyfacts/admission"
	"golang.org/x/tools/go/ssa"
)

type (
	// InputSignature admits only direct caller-supplied callback values.
	InputSignature func(value ssa.Value, method *types.Func) bool
	// CallbackSignature returns the original callback and its optional method.
	CallbackSignature func(ssa.CallInstruction) (ssa.Value, *types.Func, bool)
	callbackValue     = ssa.Value
	callbackCall      = ssa.Call
	callCommon        = ssa.CallCommon
)

var (
	_ InputSignature    = CallbackInput
	_ CallbackSignature = CallbackCall
)

func functionType(value types.Type) bool {
	_, function := value.Underlying().(*types.Signature)
	return function
}

// A callback cannot receive resource-bearing arguments in this bounded model.
// The scan is read-only and retains the existing uncharged metadata boundary.
func inertArguments(common *ssa.CallCommon) bool {
	for _, argument := range common.Args {
		if !admission.Value(argument.Type()) {
			return false
		}
	}
	return true
}
