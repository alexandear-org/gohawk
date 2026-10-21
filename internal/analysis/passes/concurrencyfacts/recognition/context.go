// Package recognition identifies symbolic context and callback operations.
// It returns exact SSA operands without resolving storage, binding callbacks,
// building summaries, or claiming that an operation completes a lifecycle.
package recognition

import (
	"github.com/kojah/gohawk/internal/analysis/passes/concurrencyfacts/admission"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// ContextKind identifies a standard context operation before identity binding.
type ContextKind uint8

const (
	ContextUnknown ContextKind = iota
	ContextEmpty
	ContextFresh
	ContextObserve
	ContextRequest
)

// OriginKind distinguishes a Done projection from a constructor result.
type OriginKind uint8

const (
	OriginUnknown OriginKind = iota
	OriginObserved
	OriginFresh
)

type (
	// ContextSignature returns the operation and the operand its engine must bind.
	ContextSignature func(*ssa.CallCommon) (ContextKind, ssa.Value)
	// OriginSignature returns a constructor call or the observed context value.
	OriginSignature func(ssa.Value) (OriginKind, *ssa.Call, ssa.Value)
	contextCall     = ssa.Call
)

var (
	_ ContextSignature                = ContextCall
	_ OriginSignature                 = ContextOrigin
	_ func(*ssa.CallCommon) ssa.Value = ssaflow.CallReceiver
)

var (
	withCancel      = syntax.PackageFunction("context", "WithCancel")
	withCancelCause = syntax.PackageFunction("context", "WithCancelCause")
	background      = syntax.PackageFunction("context", "Background")
	todo            = syntax.PackageFunction("context", "TODO")
	done            = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "context", Receiver: "Context", Name: "Done"})
)

// EmptyContext matches the standard non-canceling context constructors.
func EmptyContext(common *ssa.CallCommon) bool {
	return ssacall.CallMatchesAnySymbol(common, background, todo)
}

// CancelConstructor matches the standard constructors with explicit cancellation.
func CancelConstructor(common *ssa.CallCommon) bool {
	return ssacall.CallMatchesAnySymbol(common, withCancel, withCancelCause)
}

// DoneCall matches the standard Context.Done contract, including interface calls.
func DoneCall(common *ssa.CallCommon) bool {
	return ssacall.CallMatchesSymbol(common, done)
}

func parentArgument(common *ssa.CallCommon) ssa.Value { return common.Args[0] }

func cancelTarget(common *ssa.CallCommon) bool {
	return !common.IsInvoke() && common.Value != nil && admission.CancelFunctionType(common.Value.Type())
}
