package admission

import (
	"go/types"

	"github.com/kojah/gohawk/internal/engine/syntax"
)

// waitGroupPointer checks the exact standard-library group pointer contract.
func waitGroupPointer(value types.Type) bool {
	pointer, ok := value.Underlying().(*types.Pointer)
	if !ok {
		return false
	}
	return syntax.PointerStruct(pointer) != nil && syntax.NamedType(pointer.Elem(), "sync", "WaitGroup")
}

// SynchronizationPointer identifies pointers to modeled sync primitives.
func SynchronizationPointer(value types.Type) bool {
	return waitGroupPointer(value) || MutexPointer(value) || condPointer(value)
}

// condPointer checks the exact standard-library condition pointer contract.
func condPointer(value types.Type) bool {
	pointer, ok := value.Underlying().(*types.Pointer)
	return ok && syntax.NamedType(pointer.Elem(), "sync", "Cond")
}

// CancellationType identifies standard context and cancellation function types.
func CancellationType(value types.Type) bool {
	return syntax.NamedType(value, "context", "Context") || CancelFunctionType(value)
}

// CancelFunctionType identifies standard cancellation function types.
func CancelFunctionType(value types.Type) bool {
	return syntax.NamedType(value, "context", "CancelFunc") || syntax.NamedType(value, "context", "CancelCauseFunc")
}

// ContainsSynchronization detects copied primitive state in structs and arrays.
// Whole-owner stores can reset embedded state just as a direct primitive store
// can. Pointer fields do not embed that state.
func ContainsSynchronization(value types.Type) bool {
	if SynchronizationPointer(types.NewPointer(value)) {
		return true
	}
	switch value := value.Underlying().(type) {
	case *types.Struct:
		for field := range value.Fields() {
			if ContainsSynchronization(field.Type()) {
				return true
			}
		}
	case *types.Array:
		return ContainsSynchronization(value.Elem())
	}
	return false
}

// MutexPointer identifies standard mutex pointers, including reader/writer mutexes.
func MutexPointer(value types.Type) bool {
	pointer, ok := value.Underlying().(*types.Pointer)
	return ok && (syntax.NamedType(pointer.Elem(), "sync", "Mutex") || syntax.NamedType(pointer.Elem(), "sync", "RWMutex"))
}
