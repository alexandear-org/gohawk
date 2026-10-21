// Package valueforms recognizes one transparent SSA wrapper under an explicit
// caller-selected mask. It does not traverse values or infer transparency.
package valueforms

import "golang.org/x/tools/go/ssa"

// Form is a set of SSA transformations a caller permits for its own proof.
type Form uint8

const (
	// ChangeInterface permits interface-to-interface representation changes.
	ChangeInterface Form = 1 << iota
	// ChangeType permits SSA changes between representation-identical types.
	ChangeType
	// Convert permits conversions whose meaning the caller has checked.
	Convert
	// MakeInterface permits boxing a value into an interface.
	MakeInterface
	// TypeAssert permits extracting an asserted value, never its success flag.
	TypeAssert
)

// Signature binds the generated one-step recognition API.
type Signature func(ssa.Value, Form) (ssa.Value, bool)

func (forms Form) allows(form Form) bool { return forms&form != 0 }

func changeInterface(value ssa.Value) (ssa.Value, bool) {
	node, ok := value.(*ssa.ChangeInterface)
	if !ok {
		return nil, false
	}
	return node.X, true
}

func changeType(value ssa.Value) (ssa.Value, bool) {
	node, ok := value.(*ssa.ChangeType)
	if !ok {
		return nil, false
	}
	return node.X, true
}

func convert(value ssa.Value) (ssa.Value, bool) {
	node, ok := value.(*ssa.Convert)
	if !ok {
		return nil, false
	}
	return node.X, true
}

func makeInterface(value ssa.Value) (ssa.Value, bool) {
	node, ok := value.(*ssa.MakeInterface)
	if !ok {
		return nil, false
	}
	return node.X, true
}

func assertion(value ssa.Value) (ssa.Value, bool, bool) {
	node, ok := value.(*ssa.TypeAssert)
	if !ok {
		return nil, false, false
	}
	return node.X, node.CommaOk, true
}

func extract(value ssa.Value) (ssa.Value, int, bool) {
	node, ok := value.(*ssa.Extract)
	if !ok {
		return nil, 0, false
	}
	return node.Tuple, node.Index, true
}
