package deferinloop

import (
	"slices"
	"testing"

	"github.com/kojah/gohawk/internal/analysis/passes/lifecyclefacts"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ssa"
)

func TestResourceAcquisitionRequiresExactResult(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "acquisition", `package acquisition
import("os";"net/http")
func file(other *os.File){f,_:=os.Open("x");defer f.Close();defer other.Close()}
func body(){r,_:=http.Get("x");defer r.Body.Close()}
func replaced(){r,_:=http.Get("x");r.Body=nil;defer r.Body.Close()}
type mock struct{Body *os.File}
func (*mock) Close(){}
func create()*mock{return &mock{}}
func misleading(){m:=create();defer m.Close()}
`)
	pass := &analysis.Pass{Fset: pkg.Prog.Fset, Pkg: pkg.Pkg, ResultOf: map[*analysis.Analyzer]any{
		lifecyclefacts.Analyzer: lifecyclefacts.Summaries{},
	}}
	evidence := lifecyclefacts.NewLifecycleEvidence(pass, "deferinloop", "")
	for _, test := range []struct {
		name       string
		deferIndex int
		known      bool
	}{
		{"file", 0, true},
		{"file", 1, false},
		{"body", 0, true},
		{"replaced", 0, false},
		{"misleading", 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := pkg.Func(test.name)
			call := ssaflow.InstructionsOf[*ssa.Call](function)[0]
			deferred := ssaflow.InstructionsOf[*ssa.Defer](function)[test.deferIndex]
			target := ssaflow.CallReceiver(deferred.Common())
			methods, known := classifyResourceAcquisition(evidence, call, target)
			if known != test.known || known && !slices.Equal(methods, []string{"Close"}) || !known && methods != nil {
				t.Fatalf("%s: cleanup=%v known=%v, want known=%v", deferred, methods, known, test.known)
			}
		})
	}
}
