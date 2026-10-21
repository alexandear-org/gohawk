package operations

import "golang.org/x/tools/go/ssa"

// LeafKind selects storage for a value without a recorded instruction result.
// Pending means a local instruction has not been reached by the fixpoint yet.
type LeafKind uint8

const (
	LeafUnknown LeafKind = iota
	LeafExternal
	LeafNil
	LeafOpaque
	LeafPending
)

type leafSignature func(ssa.Value, *ssa.Function) LeafKind

var _ leafSignature = Leaf

func instruction(value ssa.Value) (ssa.Instruction, bool) { //nolint:ireturn // Preserve instruction identity.
	result, ok := value.(ssa.Instruction)
	return result, ok
}

func localInstruction(instruction ssa.Instruction, owner *ssa.Function) bool {
	return instruction.Parent() == owner
}
