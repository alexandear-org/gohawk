package lifecycleforms

import "golang.org/x/tools/go/ssa"

// RetentionKind selects the host evidence required by a use under the chosen
// retention mode. Recognition alone supplies no positive ownership claim.
type RetentionKind uint8

const (
	RetentionIgnore RetentionKind = iota
	RetentionStore
	RetentionClosure
	RetentionValue
	RetentionReturn
	RetentionCall
)

// RetentionUse retains the original node or operands for a reached proof.
// Calls, closures and stores still require the enclosing pass's evidence.
type RetentionUse struct {
	Kind        RetentionKind
	Store       *ssa.Store
	Closure     *ssa.MakeClosure
	Value       ssa.Value
	Values      []ssa.Value
	Instruction ssa.Instruction
}
type retentionSignature func(ssa.Instruction, bool, bool) RetentionUse

var _ retentionSignature = Retains

// EscapeKind distinguishes terminal escape evidence from a source that must
// itself escape or a store/call that needs the enclosing search's proof.
type EscapeKind uint8

const (
	EscapeIgnored EscapeKind = iota
	EscapeDefinite
	EscapeFollow
	EscapeStore
	EscapeCall
)

// EscapeUse supplies one continuation without walking uses or local storage.
type EscapeUse struct {
	Kind        EscapeKind
	Source      ssa.Value
	Store       *ssa.Store
	Instruction ssa.Instruction
}
type escapeSignature func(ssa.Value, ssa.Instruction, bool) EscapeUse

var _ escapeSignature = Escapes

func valueInstruction(instruction ssa.Instruction) (ssa.Value, bool) { //nolint:ireturn // Preserve exact SSA identity.
	value, ok := instruction.(ssa.Value)
	return value, ok
}

func sameValue(left, right ssa.Value) bool { return left == right }

type escapeAliasSignature func(ssa.Instruction) (ssa.Value, bool)

var _ escapeAliasSignature = EscapeAlias
