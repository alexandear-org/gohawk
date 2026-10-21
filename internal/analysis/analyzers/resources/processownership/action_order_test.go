package processownership

import (
	"testing"

	"github.com/kojah/gohawk/internal/analysis/passes/lifecyclefacts"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/ssa"
)

func TestProcessActionEvidenceOrder(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "actionorder", `package actionorder
import("os";"os/exec")
type owner struct{process *os.Process}
func wait(cmd *exec.Cmd){cmd.Wait()}
func deferred(cmd *exec.Cmd){defer func(){cmd.Wait()}()}
func release(cmd *exec.Cmd){cmd.Process.Release()}
func exported(cmd *exec.Cmd,owner *owner){owner.process=cmd.Process}
func handle(cmd *exec.Cmd){cmd.Process.Wait()}
func exit(cmd *exec.Cmd){os.Exit(0)}
func unrelated(cmd *exec.Cmd){println(1)}
func merged(cmd,other *exec.Cmd,choose bool){selected:=cmd;if choose{selected=other};selected.Wait()}
`)
	const (
		unknown   = proofs.EvidenceUnknown
		proven    = proofs.EvidenceProven
		disproven = proofs.EvidenceDisproven
	)
	for _, test := range []struct {
		name string
		pick func(*ssa.Function) ssa.Instruction
		want [4]proofs.EvidenceState
	}{
		{"wait", lastActionInstruction[*ssa.Call], [4]proofs.EvidenceState{proven, proven, proven, proven}},
		{"deferred", lastActionInstruction[*ssa.Defer], [4]proofs.EvidenceState{unknown, unknown, unknown, proven}},
		{"release", lastActionInstruction[*ssa.Call], [4]proofs.EvidenceState{unknown, unknown, proven, proven}},
		{"exported", lastActionInstruction[*ssa.Store], [4]proofs.EvidenceState{proven, proven, proven, proven}},
		{"handle", lastActionInstruction[*ssa.Call], [4]proofs.EvidenceState{unknown, unknown, proven, proven}},
		{"exit", lastActionInstruction[*ssa.Call], [4]proofs.EvidenceState{unknown, unknown, proven, proven}},
		{"unrelated", lastActionInstruction[*ssa.Call], [4]proofs.EvidenceState{unknown, unknown, unknown, disproven}},
		{"merged", lastActionInstruction[*ssa.Call], [4]proofs.EvidenceState{unknown, unknown, unknown, unknown}},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := pkg.Func(test.name)
			instruction := test.pick(function)
			for index, limit := range []int{0, 1, 5, processPoolBudget} {
				proof := &commandProof{pool: proofs.NewSearchBudget(limit), evidence: lifecyclefacts.NewLifecycleEvidence(nil, "test", "action-order")}
				got := proof.action(instruction, function.Params[0])
				t.Logf("%s limit=%d: state=%v remaining=%d", instruction, limit, got, proof.pool.Remaining())
				if got != test.want[index] {
					t.Fatalf("limit %d: got %v, want %v", limit, got, test.want[index])
				}
				if test.name == "exit" && limit == 5 && proof.pool.Remaining() != 0 {
					t.Fatal("exit fixture no longer exercises a late positive with an empty allowance")
				}
			}
		})
	}
}

func lastActionInstruction[T ssa.Instruction](function *ssa.Function) ssa.Instruction {
	instructions := ssaflow.InstructionsOf[T](function)
	return instructions[len(instructions)-1]
}

func TestProcessRemainderEvidencePrecedence(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "remainder", `package remainder
 import("os";"os/exec")
 type owner struct{process *os.Process}
 func exit(cmd *exec.Cmd){os.Exit(0)}
 func unrelated(cmd *exec.Cmd){println(1)}
 func read(cmd *exec.Cmd)*os.Process{return cmd.Process}
 func exported(cmd *exec.Cmd,owner *owner){owner.process=cmd.Process}
 `)
	for _, test := range []struct {
		name           string
		pick           func(*ssa.Function) ssa.Instruction
		fallback, want proofs.EvidenceState
		exhausted      bool
	}{
		{"exit", lastActionInstruction[*ssa.Call], decisionUnknown, decisionProven, true},
		{"unrelated", lastActionInstruction[*ssa.Call], decisionDisproven, decisionUnknown, true},
		{"read", lastActionInstruction[*ssa.UnOp], decisionUnknown, decisionUnknown, false},
		{"read", lastActionInstruction[*ssa.UnOp], decisionDisproven, decisionDisproven, false},
		{"exported", lastActionInstruction[*ssa.Store], decisionUnknown, decisionProven, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := pkg.Func(test.name)
			proof := &commandProof{pool: proofs.NewSearchBudget(0), evidence: lifecyclefacts.NewLifecycleEvidence(nil, "test", "remainder")}
			got := decideProcessRemainder(proof, test.pick(function), function.Params[0], test.fallback)
			if got != test.want || proof.pool.Exhausted() != test.exhausted {
				t.Fatalf("got state=%v exhausted=%v, want %v/%v", got, proof.pool.Exhausted(), test.want, test.exhausted)
			}
		})
	}
}
