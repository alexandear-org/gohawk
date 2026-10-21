// Package selection recognizes SSA select-dispatch comparisons. It establishes
// structure only; callers own edge feasibility and communication semantics.
package selection

import (
	"go/constant"
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// Comparison names the exact select, compared integer and comparison direction.
// Reversed retains operand order for consumers that admit only canonical SSA.
type Comparison struct {
	Select          *ssa.Select
	Number          int64
	Equal, Reversed bool
}

// ComparisonSignature is the structural recognizer generated from comparison.rules.
type ComparisonSignature func(ssa.Value) (Comparison, bool)

func comparison(value ssa.Value) (equal bool, left, right ssa.Value, matched bool) {
	binary, ok := value.(*ssa.BinOp)
	if !ok || binary.Op != token.EQL && binary.Op != token.NEQ {
		return false, nil, nil, false
	}
	return binary.Op == token.EQL, binary.X, binary.Y, true
}

func extract(value ssa.Value) (tuple ssa.Value, index int, matched bool) {
	projection, ok := value.(*ssa.Extract)
	if !ok {
		return nil, 0, false
	}
	return projection.Tuple, projection.Index, true
}

func selectValue(value ssa.Value) (*ssa.Select, bool) {
	selected, ok := value.(*ssa.Select)
	return selected, ok
}

func integer(value ssa.Value) (int64, bool) {
	literal, ok := value.(*ssa.Const)
	if !ok || literal.Value == nil || literal.Value.Kind() != constant.Int {
		return 0, false
	}
	return constant.Int64Val(literal.Value)
}

func result(selected *ssa.Select, number int64, equal, reversed bool) Comparison {
	return Comparison{Select: selected, Number: number, Equal: equal, Reversed: reversed}
}
