package cancellationownership

import (
	"testing"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
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
