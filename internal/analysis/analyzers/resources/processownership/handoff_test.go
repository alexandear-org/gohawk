package processownership

import (
	"testing"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"github.com/kojah/gohawk/internal/testsupport/analyzertest"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/ssa"
)

func TestProcessClosureChoices(t *testing.T) {
	analyzertest.Run(t, analysistest.TestData(), Analyzer(), "processchoices")
}

func TestPossibleNonreturningWaitAllowance(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "waitreach", `package waitreach
 import "os/exec"
 func endless(cmd *exec.Cmd){go func(){cmd.Wait();for{}}()}
 func returning(cmd *exec.Cmd){go func(){cmd.Wait()}()}
 func unrelated(cmd *exec.Cmd){go func(){for{}}()}
 `)
	for _, name := range []string{"endless", "returning", "unrelated"} {
		fn := pkg.Func(name)
		instruction := ssaflow.InstructionsOf[*ssa.Go](fn)[0]
		pool := proofs.NewSearchBudget(processPoolBudget)
		cut := provePossibleWaitHandoff(instruction, fn.Params[0], pool.Within(0))
		if cut.Reason != proofs.EvidenceBudgetExhausted || cut.State != proofs.EvidenceUnknown || pool.Exhausted() {
			t.Fatalf("%s cutoff: %+v", name, cut)
		}
		fresh := provePossibleWaitHandoff(instruction, fn.Params[0], pool.Within(processQueryBudget))
		want := proofs.EvidenceDisproven
		if name == "endless" {
			want = proofs.EvidenceUnknown
		}
		if fresh.State != want || fresh.Reason == proofs.EvidenceBudgetExhausted {
			t.Fatalf("%s fresh: %+v", name, fresh)
		}
		if name == "endless" && (fresh.Method != "Wait" || fresh.Provenance != proofs.EvidenceFromLocalSSA) {
			t.Fatalf("nonreturning waiter lost its completion payload: %+v", fresh)
		}
	}
}

func TestImmediateWaitHandoffFormsDoNotSpend(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "immediatehandoff", `package immediatehandoff
import "os/exec"
func helper(cmd *exec.Cmd){cmd.Wait()}
func direct(cmd *exec.Cmd){cmd.Wait()}
func synchronous(cmd *exec.Cmd){helper(cmd)}
func plain(cmd *exec.Cmd){}
`)
	for _, name := range []string{"direct", "synchronous", "plain"} {
		function := pkg.Func(name)
		instruction := function.Blocks[0].Instrs[0]
		for _, alreadySpent := range []bool{false, true} {
			budget := proofs.NewSearchBudget(0)
			if alreadySpent {
				budget.Spend()
			}
			got := provePossibleWaitHandoff(instruction, function.Params[0], budget)
			want := proofs.Proof{State: proofs.EvidenceDisproven, Reason: proofs.EvidenceNotFound}
			if got != want || budget.Exhausted() != alreadySpent {
				t.Fatalf("%s spent=%v: proof=%+v exhausted=%v", name, alreadySpent, got, budget.Exhausted())
			}
		}
	}
}
