package recognition

import (
	"go/token"
	"go/types"

	"github.com/kojah/gohawk/internal/analysis/passes/concurrencyfacts/admission"

	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// ReferenceKind selects how an already resolved resource enters the engine's
// identity model. Rejected calls retain their original value on failure.
type ReferenceKind uint8

const (
	ReferenceUnknown ReferenceKind = iota
	ReferenceExact
	ReferenceAddress
	ReferenceRejected
)

// AddressKind separates exact globals, captured cells and field loads from
// ordinary formal roots. Blocked unary forms cannot fall through to binding.
type AddressKind uint8

const (
	AddressBlocked AddressKind = iota
	AddressBinding
	AddressGlobal
	AddressCapture
	AddressField
)

type (
	// ReferenceSignature classifies storage-resolved values before path queries.
	ReferenceSignature func(ssa.Value) (ReferenceKind, ssa.Value, bool)
	// RootSignature selects how a callee root is mapped into its caller.
	RootSignature func(ssa.Value) (AddressKind, *ssa.FreeVar, *ssa.FieldAddr)
	// FixedLoadSignature requires a load before selecting its captured/field source.
	FixedLoadSignature func(*ssa.UnOp) (AddressKind, *ssa.FreeVar, *ssa.FieldAddr)
	// PathRootSignature admits direct roots or returns the unary root to prove.
	PathRootSignature func(ssa.Value, bool) (bool, *ssa.UnOp)
	referenceValue    = ssa.Value
	referenceCall     = ssa.Call
)

var (
	_       func(types.Type) bool = admission.MutexPointer
	_       ReferenceSignature    = ResourceReference
	_       RootSignature         = BindingRoot
	_       FixedLoadSignature    = FixedLoad
	_       PathRootSignature     = PathRoot
	newCond                       = syntax.PackageFunction("sync", "NewCond")
)

const dereferenceToken = token.MUL

// ConditionConstructor matches sync.NewCond by declaration identity.
func ConditionConstructor(common *ssa.CallCommon) bool {
	return ssacall.CallMatchesSymbol(common, newCond)
}
