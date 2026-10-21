// Package operations recognizes heap value definitions without reading or
// changing the region graph. Plans preserve the original SSA values; the graph
// applies projection, load and alias effects under its existing memory policy.
package operations

import (
	"go/token"

	"github.com/kojah/gohawk/internal/engine/ssaflow"

	"golang.org/x/tools/go/ssa"
)

// DefinitionKind identifies the memory interpretation of a value definition.
type DefinitionKind uint8

const (
	Ignored DefinitionKind = iota
	Alias
	FieldAddress
	FieldLoad
	IndexAddress
	IndexLoad
	Sliced
	Load
	Opaque
)

// Definition retains exact operands for the graph to interpret. Only the
// fields selected by Kind are populated; no alias or storage search occurs here.
type Definition struct {
	Kind   DefinitionKind
	Value  ssa.Value
	Source ssa.Value
	Index  ssa.Value
	Field  int
	Slice  *ssa.Slice
}

type definitionSignature func(ssa.Value) Definition

var _ definitionSignature = Define

const loadToken = token.MUL

func wrapper(value ssa.Value) (ssa.Value, bool) { //nolint:ireturn // Exact original operand.
	forms := ssaflow.TransparentChangeInterface | ssaflow.TransparentChangeType | ssaflow.TransparentConvert | ssaflow.TransparentMakeInterface
	return ssaflow.UnwrapTransparentValue(value, forms)
}
