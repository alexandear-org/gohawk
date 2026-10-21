package completionmapping

import (
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"golang.org/x/tools/go/ssa"
)

// StorageOwner returns the root of target's static field or element
// path when that root is argument or proven to be the same value, and nil
// otherwise. The target is storage beneath the owner, not its stored value;
// a matching cleanup must select the same path, never a sibling field.
// https://github.com/ovn-kubernetes/libovsdb/blob/6acd868996b9393b932a1eeeec1ea4e6c722ebe8/client/client.go#L286-L299
func StorageOwner(target, argument ssa.Value, budget *proofs.SearchBudget) ssa.Value { //nolint:ireturn // SSA values keep their concrete forms.
	switch target.(type) {
	case *ssa.FieldAddr, *ssa.IndexAddr:
		if ssaflow.ValueIsAccessPathFromWithin(target, argument, budget) {
			return argument
		}
	default:
		return nil
	}
	root := target
	for {
		if !budget.Spend() {
			return nil
		}
		switch address := root.(type) {
		case *ssa.FieldAddr:
			root = address.X
			continue
		case *ssa.IndexAddr:
			root = address.X
			continue
		}
		break
	}
	if root == target || !ssaflow.ValueIsAccessPathFromWithin(target, root, budget) ||
		root != argument && !heapmodel.DefinitelySameValue(root, argument) {
		return nil
	}
	if budget.Exhausted() || budget.PoolExhausted() {
		return nil
	}
	return root
}

// ExactReceiver accepts only the original transparent wrappers around the
// parameter, charging every step. A load or projection remains opaque.
func ExactReceiver(receiver, parameter ssa.Value, budget *proofs.SearchBudget) bool {
	if receiver == nil || parameter == nil || !budget.Spend() {
		return false
	}
	if inner, ok := ssaflow.UnwrapTransparentValue(
		receiver, ssaflow.TransparentChangeInterface|ssaflow.TransparentChangeType|ssaflow.TransparentConvert|ssaflow.TransparentMakeInterface,
	); ok {
		return ExactReceiver(inner, parameter, budget)
	}
	return receiver == parameter
}
