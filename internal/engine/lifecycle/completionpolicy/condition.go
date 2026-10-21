package completionpolicy

import (
	"go/constant"
	"go/token"

	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"

	"golang.org/x/tools/go/ssa"
)

// ConditionKind distinguishes a known nilness outcome from a Boolean operand
// that still needs the caller's negation and nested-comparison decoding.
type ConditionKind uint8

const (
	ConditionOpaque ConditionKind = iota
	ConditionOutcome
	ConditionFollow
)

// ConditionStep selects one conditional-completion operation. It establishes
// no cleanup or feasible path until the caller applies the original search.
type ConditionStep struct {
	Kind    ConditionKind
	Value   ssa.Value
	Outcome ssacall.Outcome
	Truth   bool
}

type conditionSignature func(*ssa.BinOp, bool) ConditionStep

var _ conditionSignature = RecognizeCondition

const (
	conditionEqualToken    = token.EQL
	conditionNotEqualToken = token.NEQ
)

func errorNilComparison(literal *ssa.Const, operand ssa.Value) bool {
	return literal.IsNil() && syntax.IsErrorType(operand.Type())
}

func nonconstantOperand(value ssa.Value) bool {
	_, constantOperand := value.(*ssa.Const)
	return !constantOperand
}

func conditionBoolean(value constant.Value) (bool, bool) {
	if value == nil || value.Kind() != constant.Bool {
		return false, false
	}
	return constant.BoolVal(value), true
}

func nilComparisonOutcome(op token.Token, truth bool) ssacall.Outcome {
	if truth == (op == token.EQL) {
		return ssacall.OutcomeNil
	}
	return ssacall.OutcomeNonNil
}

func comparisonOperandTruth(op token.Token, truth, literal bool) bool {
	return (truth == (op == token.EQL)) == literal
}
