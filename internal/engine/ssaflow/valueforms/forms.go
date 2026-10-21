// Package valueforms recognizes one SSA source form. Transparent wrappers use
// an explicit caller-selected mask; ownership forms expose a holder candidate.
// Neither API traverses values or establishes ownership by itself.
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
