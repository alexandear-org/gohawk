// Package lockidentity names mutex instances from exact storage and SSA origin.
// Unknown call results and interrupted storage queries supply no identity;
// declarations alone identify classes, not distinct runtime lock instances.
package lockidentity

import (
	"go/types"

	"github.com/kojah/gohawk/internal/engine/heapmodel"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"golang.org/x/tools/go/ssa"
)

// Within shares reaching and observation-time storage with the
// enclosing request. Cutoff supplies no identity, never a fallback slot name.
func Within(value ssa.Value, budget *proofs.SearchBudget) string {
	name := identity(ssaflow.NewReachingWalk(ssaflow.TransparentNone).Within(budget), value, budget)
	if budget.Exhausted() || budget.PoolExhausted() {
		return ""
	}
	return name
}

func identity(walk ssaflow.ReachingWalk, value ssa.Value, budget *proofs.SearchBudget) string {
	if value == nil || !walk.Mark(value) {
		return ""
	}
	resolved := heapmodel.NewStorage(budget).Resolve(value)
	if budget.Exhausted() || budget.PoolExhausted() {
		return ""
	}
	if resolved.Proven() && resolved.Value != value {
		return identity(walk, resolved.Value, budget)
	}
	if source, ok := ssaflow.IdentitySource(value); ok {
		return identity(walk, source, budget)
	}
	return decideIdentityLeaf(walk, value, budget)
}

// Getter recognizes only a pointer receiver returning the address of one
// direct field. Loads, branches, calls and side effects are intentionally opaque;
// in particular a pointer-valued field could change between calls. This is the
// body of Account.Mu, not a convention attached to its name:
// https://github.com/james-6-23/codex2api/blob/4f96afe95bb16132347f4ab74e63b0b1fa0f778b/auth/store.go#L553-L555
func Getter(call *ssa.Call) (ssa.Value, *types.Var) { //nolint:ireturn // The owner is an SSA value.
	callee := call.Common().StaticCallee()
	if callee == nil || callee.Signature.Recv() == nil || len(callee.Params) != 1 ||
		len(call.Common().Args) != 1 || len(callee.Blocks) != 1 {
		return nil, nil
	}
	if _, ok := callee.Params[0].Type().Underlying().(*types.Pointer); !ok {
		return nil, nil
	}
	instructions := callee.Blocks[0].Instrs
	if len(instructions) != 2 {
		return nil, nil
	}
	field, ok := instructions[0].(*ssa.FieldAddr)
	if !ok || field.X != callee.Params[0] {
		return nil, nil
	}
	returned, ok := instructions[1].(*ssa.Return)
	if !ok || len(returned.Results) != 1 || returned.Results[0] != field {
		return nil, nil
	}
	return call.Common().Args[0], Field(field.X.Type(), field.Field)
}

func addressIdentity(walk ssaflow.ReachingWalk, fieldAddress *ssa.FieldAddr, budget *proofs.SearchBudget) (string, bool) {
	field := Field(fieldAddress.X.Type(), fieldAddress.Field)
	if field == nil {
		return "", false
	}
	if owner := identity(walk, fieldAddress.X, budget); owner != "" {
		return owner + "." + field.Name(), true
	}
	// A declaration identifies a lock class, not an instance. When a call
	// returns an unknown owner, its field must remain unknown too: a pool can
	// return a different container on each loop iteration.
	// https://github.com/encodeous/nylon/blob/c4a96c804f7aa08512721dec7994907eab100bc8/polyamide/device/receive.go#L176-L180
	return "", false
}

// A value receiver can be copied into a local SSA slot and reloaded for each
// field access. Only an exact aggregate read back to the immutable parameter
// or capture identifies those copies as the same field. Partial/conflicting
// writes leave the read unresolved, so they cannot equate different locks.
// https://github.com/tikv/client-go/blob/b9fc0b7719d3ea62bd9904cd31fbd27715ff08bc/txnkv/transaction/pessimistic.go#L489-L518
func copiedIdentity(walk ssaflow.ReachingWalk, fieldValue *ssa.Field, budget *proofs.SearchBudget) (string, bool) {
	field := Field(fieldValue.X.Type(), fieldValue.Field)
	if field == nil {
		return "", false
	}
	source := heapmodel.NewStorage(budget).Resolve(fieldValue.X)
	if !source.Proven() {
		return "", false
	}
	switch source.Value.(type) {
	case *ssa.Parameter, *ssa.FreeVar:
		if owner := identity(walk, source.Value, budget); owner != "" {
			return owner + "." + field.Name(), true
		}
	}
	return "", false
}

func indexedIdentity(walk ssaflow.ReachingWalk, ownerValue, indexValue ssa.Value, budget *proofs.SearchBudget) (string, bool) {
	owner := identity(walk, ownerValue, budget)
	index := identity(walk, indexValue, budget)
	if owner == "" || index == "" {
		return "", false
	}
	return owner + "[" + index + "]", true
}

// Field resolves one direct struct member, optionally behind a pointer.
func Field(value types.Type, index int) *types.Var {
	if pointer, ok := value.Underlying().(*types.Pointer); ok {
		value = pointer.Elem()
	}
	structure, ok := value.Underlying().(*types.Struct)
	if !ok || index < 0 || index >= structure.NumFields() {
		return nil
	}
	return structure.Field(index)
}

// Getter results must remain terminal: an unknown call cannot fall back to its
// SSA value name and invent a stable held lock across loop iterations.
func getterIdentity(walk ssaflow.ReachingWalk, call *ssa.Call, budget *proofs.SearchBudget) (string, bool) {
	owner, field := Getter(call)
	if field == nil {
		return "", false
	}
	if name := identity(walk, owner, budget); name != "" {
		return name + "." + field.Name(), true
	}
	return "", false
}

type leafSignature func(ssaflow.ReachingWalk, ssa.Value, *proofs.SearchBudget) string

var _ leafSignature = decideIdentityLeaf

func globalName(value *ssa.Global) string       { return value.Name() }
func parameterName(value *ssa.Parameter) string { return value.Parent().String() + "." + value.Name() }

func freeName(value *ssa.FreeVar) string { return value.Parent().String() + ":free:" + value.Name() }

func allocationName(value *ssa.Alloc) string {
	return value.Parent().String() + ":local:" + value.Comment + ":" + value.Name()
}
func hasLiteral(value *ssa.Const) bool    { return value.Value != nil }
func literalName(value *ssa.Const) string { return "constant:" + value.Value.ExactString() }
func valueName(value ssa.Value) string {
	if parent := value.Parent(); parent != nil && value.Name() != "" {
		return parent.String() + ":value:" + value.Name()
	}
	return ""
}
