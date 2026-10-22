package cancellationownership

import (
	"testing"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/ssa"
)

func TestCancellationLabelQueriesKeepCutoffPrecedence(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "labels", `package labels
 func invoke(cancel func()){cancel()}
 func direct(cancel func()){cancel()}
 func helper(cancel func()){invoke(cancel)}
 func launched(cancel func()){go invoke(cancel)}
 func closure(cancel func())func(){return func(){cancel()}}
 `)
	cases := []struct {
		name        string
		instruction ssa.Instruction
		action      cancellationAction
		reason      cancellationReason
	}{
		{"direct", ssaflow.InstructionsOf[*ssa.Call](pkg.Func("direct"))[0], cancellationActionRelease, reasonLabelRelease},
		{"helper", ssaflow.InstructionsOf[*ssa.Call](pkg.Func("helper"))[0], cancellationActionUnknown, reasonLabelHelperUndecided},
		{"launched", ssaflow.InstructionsOf[*ssa.Go](pkg.Func("launched"))[0], cancellationActionUnknown, reasonLabelLaunchedHelper},
		{"closure", ssaflow.InstructionsOf[*ssa.MakeClosure](pkg.Func("closure"))[0], cancellationActionNone, reasonCancellationNone},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			classifier := &cancellationClassifier{cancel: pkg.Func(test.name).Params[0], pool: proofs.NewSearchBudget(0)}
			label := classifier.classifyAction(test.instruction)
			if label.action != test.action || label.reason != test.reason {
				t.Fatalf("%s: label=%+v want %v/%v", test.instruction, label, test.action, test.reason)
			}
		})
	}
}

func TestParentOriginRequiresExactResultAndFrame(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "parentorigin", `package parentorigin
import "context"
func subject(){p,c:=context.WithCancel(context.Background());_,d:=context.WithCancel(p);defer c();defer d()}
func other(){_,c:=context.WithCancel(context.Background());defer c()}
`)
	var calls []*ssa.Call
	for _, call := range ssaflow.InstructionsOf[*ssa.Call](pkg.Func("subject")) {
		if _, recognized := cancellationContractFor(call.Common()); recognized {
			calls = append(calls, call)
		}
	}
	if len(calls) != 2 {
		t.Fatalf("fixture has %d cancellation constructors", len(calls))
	}
	parent, child := calls[0], calls[1]
	value, cancel := ssacall.CallResult(parent, 0), ssacall.CallResult(parent, 1)
	got := classifyParentOrigin(value, child, nil)
	if got == nil || got.cancel != cancel || got.actions == nil || got.parent != nil {
		t.Fatalf("exact parent result = %+v", got)
	}
	foreign := ssaflow.InstructionsOf[*ssa.Call](pkg.Func("other"))[1]
	for name, input := range map[string]ssa.Value{
		"cancel result": cancel,
		"wrapped":       &ssa.ChangeInterface{X: value},
		"merged":        &ssa.Phi{},
	} {
		if classifyParentOrigin(input, child, nil) != nil {
			t.Errorf("%s supplied an exact parent", name)
		}
	}
	if classifyParentOrigin(value, foreign, nil) != nil {
		t.Fatal("parent from another function supplied a cancellation owner")
	}
}

func TestNonliteralDeferDoesNotAllocateCancellationBudget(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "nonliteral", `package nonliteral
func invoke(cancel func()){cancel()}
func subject(cancel func()){defer invoke(cancel)}
`)
	deferred := ssaflow.InstructionsOf[*ssa.Defer](pkg.Func("subject"))[0]
	var classifier *cancellationClassifier
	label, recognized := classifier.deferredLiteralLabel(deferred)
	if recognized || label != (cancellationLabel{}) {
		t.Fatalf("nonliteral defer = %+v/%v", label, recognized)
	}
}
