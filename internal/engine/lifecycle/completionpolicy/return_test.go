package completionpolicy_test

import (
	"go/token"
	"testing"

	"github.com/kojah/gohawk/internal/engine/lifecycle/completionpolicy"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/ssa"
)

const returnFixture = `package returncases
 type failure struct{}
 func (*failure) Error()string{return "failure"}
 func no()bool{return false}
 func yes()bool{return true}
 func unknown(value bool)bool{return value}
 func typedNil()error{var value *failure;return value}
 func pair()(int,bool){return 1,true}
 func forward()(int,bool){return pair()}
 var boolSink bool
 var errorSink error
 func booleanSpill(value bool)bool{defer func(){boolSink = value}();return value}
 func errorSpill(value error)error{defer func(){errorSink = value}();return value}
`

func TestReturnPredicatePrecedesEarlierCompletion(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "returncases", returnFixture)
	for _, test := range []struct {
		function  string
		result    int
		outcome   ssacall.Outcome
		completed bool
		kind      completionpolicy.ReturnKind
	}{
		{"no", 0, ssacall.OutcomeTrue, true, completionpolicy.ReturnIrrelevant},
		{"yes", 0, ssacall.OutcomeTrue, false, completionpolicy.ReturnUncovered},
		{"yes", 0, ssacall.OutcomeTrue, true, completionpolicy.ReturnCovered},
		{"unknown", 0, ssacall.OutcomeTrue, true, completionpolicy.ReturnCovered},
		{"yes", 1, ssacall.OutcomeTrue, true, completionpolicy.ReturnUncovered},
		{"typedNil", 0, ssacall.OutcomeNil, true, completionpolicy.ReturnIrrelevant},
		{"typedNil", 0, ssacall.OutcomeNonNil, true, completionpolicy.ReturnCovered},
	} {
		returned := ssaflow.InstructionsOf[*ssa.Return](pkg.Func(test.function))[0]
		condition := ssacall.CallCondition{Result: test.result, Outcome: test.outcome}
		got := completionpolicy.RecognizeReturn(returned, condition, test.completed, nil)
		if got.Kind != test.kind || got.Call != nil || got.Result != 0 {
			t.Fatalf("%+v: got %+v", test, got)
		}
	}
}

func TestForwardedReturnKeepsItsResultSlot(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "returncases", returnFixture)
	function := pkg.Func("forward")
	returned := ssaflow.InstructionsOf[*ssa.Return](function)[0]
	call := ssaflow.InstructionsOf[*ssa.Call](function)[0]
	condition := ssacall.CallCondition{Result: 1, Outcome: ssacall.OutcomeTrue}
	got := completionpolicy.RecognizeReturn(returned, condition, false, nil)
	if got.Kind != completionpolicy.ReturnForward || got.Call != call || got.Result != 1 {
		t.Fatalf("forwarded result changed: %+v", got)
	}
}

func TestOnlyBooleanReturnSpillsSpendStorageAllowance(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "returncases", returnFixture)
	for _, test := range []struct {
		name      string
		outcome   ssacall.Outcome
		exhausted bool
	}{
		{"booleanSpill", ssacall.OutcomeTrue, true},
		{"errorSpill", ssacall.OutcomeNil, false},
	} {
		function := pkg.Func(test.name)
		returned := ssaflow.InstructionsOf[*ssa.Return](function)[0]
		value, ok := returned.Results[0].(*ssa.UnOp)
		if !ok || value.Op != token.MUL {
			t.Fatalf("%s did not retain its captured spill: %v", test.name, returned)
		}
		budget := proofs.NewSearchBudget(0)
		got := completionpolicy.ResolveReturnValue(value, test.outcome, budget)
		if got != value || budget.Exhausted() != test.exhausted {
			t.Fatalf("%s returned %v, exhausted=%v", test.name, got, budget.Exhausted())
		}
	}
}
