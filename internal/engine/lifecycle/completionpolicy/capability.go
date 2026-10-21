package completionpolicy

import "golang.org/x/tools/go/ssa"

// CapabilityKind selects a possible retained callback source. It establishes
// neither exact callback identity nor that the callback is ever invoked.
type CapabilityKind uint8

const (
	CapabilityAbsent CapabilityKind = iota
	CapabilityClosure
	CapabilityStorage
	CapabilityArguments
)

type capabilitySignature func(ssa.Value) (CapabilityKind, ssa.Value, []ssa.Value)

// RetainedCallback recognizes one leaf. The caller owns reaching-value folds,
// storage searches and completion queries, including their shared allowance.
var _ capabilitySignature = RetainedCallback
