package completionpolicy

import (
	"go/types"
	"slices"

	"golang.org/x/tools/go/ssa"
)

// Context admission rejects callback escapes before collecting invocation
// frames. Finding one safe invocation cannot hide another opaque consumer.
// The bounded type/address walks retain their original conservative polarity.

type contextSafeSignature func(ssa.Instruction) bool

var _ contextSafeSignature = ContextInstructionSafe

// CallbackStorageType reports whether a type may carry a callback through
// pointers or aggregate elements. Reaching the depth cap remains possible.
func CallbackStorageType(t types.Type) bool {
	for range 8 {
		switch underlying := t.Underlying().(type) {
		case *types.Signature:
			return true
		case *types.Pointer:
			t = underlying.Elem()
		case *types.Slice:
			t = underlying.Elem()
		case *types.Array:
			t = underlying.Elem()
		default:
			return false
		}
	}
	return true
}

func localCallbackAddress(value ssa.Value) bool {
	for range 8 {
		switch address := value.(type) {
		case *ssa.Alloc:
			return true
		case *ssa.IndexAddr:
			value = address.X
		case *ssa.Slice:
			value = address.X
		default:
			return false
		}
	}
	return false
}

func callbackValue(value ssa.Value) bool { return CallbackStorageType(value.Type()) }

func returnedCallback(returned *ssa.Return) bool {
	return slices.ContainsFunc(returned.Results, callbackValue)
}

func nonlocalCallbackAddress(address ssa.Value) bool { return !localCallbackAddress(address) }
