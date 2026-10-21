package completionpolicy

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// BindingKind selects the next exact callback-binding operation. An opaque
// value cannot establish callback identity or completion.
type BindingKind uint8

const (
	BindingOpaque BindingKind = iota
	BindingExact
	BindingField
	BindingElement
	BindingLoad
	BindingCell
)

// BindingStep keeps field and element payloads typed. Environment substitution,
// wrapper traversal and storage observation remain with the caller's search.
type BindingStep struct {
	Kind    BindingKind
	Source  ssa.Value
	Field   *ssa.FieldAddr
	Element *ssa.IndexAddr
}

// AggregateKind selects an unchanged aggregate root or a whole-slice step.
type AggregateKind uint8

const (
	AggregateOpaque AggregateKind = iota
	AggregateRoot
	AggregateFollow
	AggregateBounded
)

// EnclosingKind selects reference resolution in an enclosing invocation frame.
// This deliberately differs from exact callback binding's transparent forms.
type EnclosingKind uint8

const (
	EnclosingOpaque EnclosingKind = iota
	EnclosingExact
	EnclosingField
	EnclosingCell
)

// EnclosingStep retains the load as the storage observation point. A field
// lookup and a stable local cell are separate proof operations.
type EnclosingStep struct {
	Kind  EnclosingKind
	Field *ssa.FieldAddr
	Cell  *ssa.Alloc
	Load  *ssa.UnOp
}

type (
	bindingSignature   func(ssa.Value) BindingStep
	aggregateSignature func(ssa.Value) (AggregateKind, ssa.Value)
	enclosingSignature func(ssa.Value) EnclosingStep
)

var (
	_ bindingSignature   = RecognizeBinding
	_ aggregateSignature = RecognizeAggregate
	_ enclosingSignature = RecognizeEnclosing
)

const bindingLoadToken = token.MUL
