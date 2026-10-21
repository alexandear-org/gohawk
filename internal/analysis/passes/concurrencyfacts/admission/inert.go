// Package admission owns which ordinary data can be omitted from synchronization
// summaries. It recognizes instruction forms and resource-bearing types; it
// performs no SSA traversal or lifecycle search.
package admission

import (
	"go/types"

	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// Inert data is ordinary program state that cannot block, synchronize, or
// become a resource identity: a counter, flag, name, logger, or map of such
// values, including one reached through a caller's pointer. Such storage may change under other goroutines, so an
// inert value never names a channel, primitive, or context. Admitting these
// instructions omits no synchronization effect. Any later use of an inert value
// is still classified on its own, so a dynamic call, type assertion, or
// unmodeled callee keeps stopping the summary. Field addresses, reads, and
// container operations can panic on a nil owner, a bad index, or a nil map,
// which the engine already accepts for field addresses on parameters: a path
// that panics never reaches a later wait.

// Value rejects every type that could carry a resource identity directly.
// A value that embeds a primitive is rejected too, since copying it would copy
// lock state.
func Value(value types.Type) bool {
	if _, channel := value.Underlying().(*types.Chan); channel {
		return false
	}
	return !ContainsSynchronization(value) && !SynchronizationPointer(value) && !CancellationType(value)
}

// Address reports whether address points to inert data.
func Address(address ssa.Value) bool {
	pointer, ok := address.Type().Underlying().(*types.Pointer)
	return ok && Value(pointer.Elem())
}

func nilConstant(value ssa.Value) bool {
	constant, ok := value.(*ssa.Const)
	return ok && constant.IsNil()
}

// lookupElement returns the value a map lookup or type assertion produces,
// ignoring the CommaOk Boolean.
func lookupElement(value ssa.Value) types.Type {
	if tuple, ok := value.Type().(*types.Tuple); ok {
		return tuple.At(0).Type()
	}
	return value.Type()
}

// Builtin admits operations that read lengths, compare scalars, or copy inert
// elements without blocking, synchronizing, or publishing a resource.
func Builtin(common *ssa.CallCommon) bool {
	builtin, ok := common.Value.(*ssa.Builtin)
	if !ok {
		return false
	}
	switch builtin.Name() {
	case "len", "cap", "min", "max":
		return true
	case "append", "copy":
		return Elements(common.Args[0].Type()) && (len(common.Args) < 2 || Elements(common.Args[1].Type()))
	case "delete":
		return Value(common.Args[1].Type())
	default:
		return false
	}
}

// Atomic applies the sync/atomic package contract: its functions and
// methods are atomic memory accesses that never block and take no lock, so
// they synchronize nothing this model tracks. Most of them have no Go body to
// summarize, being assembly or compiler intrinsics. The operation is inert
// only when the cell it updates and every value it moves are inert: storing a
// channel or loading a pointer to a mutex would move a resource through
// memory, which the summary cannot follow.
func Atomic(common *ssa.CallCommon) bool {
	callee := common.StaticCallee()
	if callee == nil || len(common.Args) == 0 {
		return false
	}
	if origin := callee.Origin(); origin != nil {
		callee = origin
	}
	if !syntax.DeclaredInPackage(callee.Object(), "sync/atomic") || !Address(common.Args[0]) {
		return false
	}
	for _, argument := range common.Args[1:] {
		if !Operand(argument) {
			return false
		}
	}
	for result := range common.Signature().Results().Variables() {
		if !Value(result.Type()) {
			return false
		}
	}
	return true
}

// Operand checks a value an atomic operation stores. An interface, as
// atomic.Value stores, is inert only when the boxing is visible and boxes an
// inert value; an interface from elsewhere may carry a channel.
func Operand(value ssa.Value) bool {
	if !types.IsInterface(value.Type()) {
		return Value(value.Type())
	}
	boxed, ok := value.(*ssa.MakeInterface)
	return ok && Value(boxed.X.Type())
}

// Elements checks slice elements, or a nonslice value, for inert data.
func Elements(value types.Type) bool {
	if slice, ok := value.Underlying().(*types.Slice); ok {
		return Value(slice.Elem())
	}
	return Value(value)
}
