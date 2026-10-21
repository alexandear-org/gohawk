package deferinloop

import (
	"testing"

	"github.com/kojah/gohawk/internal/analysis/passes/lifecyclefacts"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	analysisTrace "github.com/kojah/gohawk/internal/reporting/trace"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ssa"
)

func TestResourceUseDecisionPrecedence(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "use", `package use
 type resource struct{}
 func (*resource) Close() {}
 func (*resource) Inspect() {}
 func consume(*resource) {}
 var retained *resource
 func subject(value *resource) { value.Inspect(); value.Close(); consume(value); retained = value }
 `)
	fn := pkg.Func("subject")
	pass := &analysis.Pass{Fset: pkg.Prog.Fset, Pkg: pkg.Pkg, ResultOf: map[*analysis.Analyzer]any{
		lifecyclefacts.Analyzer: lifecyclefacts.Summaries{},
	}}
	evidence := lifecyclefacts.NewLifecycleEvidence(pass, "deferinloop", "")
	obligation := deferObligation{target: fn.Params[0], cleanup: []string{"Close"}}
	for _, call := range ssaflow.InstructionsOf[*ssa.Call](fn) {
		want, reason := resourceLive, reasonNone
		switch ssaflow.CallName(call.Common()) {
		case "Close":
			want, reason = resourceSettled, reasonExplicitCleanup
		case "consume":
			want, reason = resourceUnknown, reasonUnsummarizedCalleeUse
		}
		got, why := classifyValueUse(evidence, analysisTrace.Probe{}, call, obligation)
		if got != want || why != reason {
			t.Errorf("%s: got %v/%v, want %v/%v", call, got, why, want, reason)
		}
	}
	stores := ssaflow.InstructionsOf[*ssa.Store](fn)
	if len(stores) != 1 {
		t.Fatalf("got %d stores, want global retention", len(stores))
	}
	got, why := classifyValueUse(evidence, analysisTrace.Probe{}, stores[0], obligation)
	if got != resourceSettled || why != reasonResourceTransferred {
		t.Fatalf("global transfer lost to opaque storage: %v/%v", got, why)
	}
}
