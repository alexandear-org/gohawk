package completionsteps_test

import (
	"go/token"
	"testing"

	"github.com/kojah/gohawk/internal/analysis/analyzers/concurrency/goroutineownership/completionsteps"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/ssa"
)

func TestCompletionInstructionModesRemainDistinct(t *testing.T) {
	for _, test := range []struct {
		name        string
		instruction ssa.Instruction
		terminal    completionsteps.TerminalStep
		tail        completionsteps.TailStep
	}{
		{"return", &ssa.Return{}, completionsteps.Returned, completionsteps.Passive},
		{"jump", &ssa.Jump{}, completionsteps.Follow, completionsteps.Passive},
		{"run defers", &ssa.RunDefers{}, completionsteps.RunDefers, completionsteps.Deferred},
		{"debug ref", &ssa.DebugRef{}, completionsteps.Reject, completionsteps.Passive},
		{"load", &ssa.UnOp{Op: token.MUL}, completionsteps.Reject, completionsteps.Passive},
		{"receive", &ssa.UnOp{Op: token.ARROW}, completionsteps.Reject, completionsteps.Work},
		{"not", &ssa.UnOp{Op: token.NOT}, completionsteps.Reject, completionsteps.Work},
		{"send", &ssa.Send{}, completionsteps.Reject, completionsteps.Work},
		{"launch", &ssa.Go{}, completionsteps.Reject, completionsteps.Work},
		{"defer", &ssa.Defer{}, completionsteps.Reject, completionsteps.Work},
		{"arithmetic", &ssa.BinOp{Op: token.ADD}, completionsteps.Reject, completionsteps.Work},
		{"branch", &ssa.If{}, completionsteps.Reject, completionsteps.Work},
		{"absent", nil, completionsteps.Reject, completionsteps.Work},
	} {
		t.Run(test.name, func(t *testing.T) {
			terminal, tail := completionsteps.Terminal(test.instruction), completionsteps.Tail(test.instruction)
			if terminal != test.terminal || tail != test.tail {
				t.Fatalf("terminal=%v tail=%v, want %v/%v", terminal, tail, test.terminal, test.tail)
			}
		})
	}
}

func TestOnlyBuiltinCloseIsPassiveTailWork(t *testing.T) {
	for _, test := range []struct {
		name, source string
		tail         completionsteps.TailStep
	}{
		{"builtinclose", `package builtinclose;func caller(done chan int){close(done)}`, completionsteps.Passive},
		{"namedclose", `package namedclose;func close(chan int){};func caller(done chan int){close(done)}`, completionsteps.Work},
	} {
		t.Run(test.name, func(t *testing.T) {
			pkg := ssaflowtest.BuildPackage(t, test.name, test.source)
			calls := ssaflow.InstructionsOf[*ssa.Call](pkg.Func("caller"))
			if len(calls) != 1 {
				t.Fatalf("calls=%d", len(calls))
			}
			if terminal, tail := completionsteps.Terminal(calls[0]), completionsteps.Tail(calls[0]); terminal != completionsteps.Reject || tail != test.tail {
				t.Fatalf("terminal=%v tail=%v", terminal, tail)
			}
		})
	}
}

func TestNotificationQueriesOnlyTerminalSends(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "notifications", `package notifications
func sent(done chan int) { done <- 1 }
func detached(done chan int) { go close(done) }
func immediate(done chan int) { close(done) }
func repeated(done chan int, again bool) { for again { close(done) } }
func deferred(done chan int, again bool) { for again { defer close(done) } }
`)
	for _, terminal := range []bool{false, true} {
		function := pkg.Func("sent")
		send := ssaflow.InstructionsOf[*ssa.Send](function)[0]
		calls := 0
		got := completionsteps.Notification(send, func(instruction ssa.Instruction) bool {
			calls++
			if instruction != send {
				t.Fatal("coverage queried another instruction")
			}
			return terminal
		})
		if calls != 1 || (got == send.Chan) != terminal || !terminal && got != nil {
			t.Fatalf("terminal=%t channel=%v calls=%d", terminal, got, calls)
		}
	}
	for _, name := range []string{"detached", "immediate", "repeated", "deferred"} {
		t.Run(name, func(t *testing.T) {
			function := pkg.Func(name)
			var instruction ssa.Instruction
			switch name {
			case "detached":
				instruction = ssaflow.InstructionsOf[*ssa.Go](function)[0]
			case "deferred":
				instruction = ssaflow.InstructionsOf[*ssa.Defer](function)[0]
			default:
				instruction = ssaflow.InstructionsOf[*ssa.Call](function)[0]
			}
			got := completionsteps.Notification(instruction, func(ssa.Instruction) bool {
				t.Fatal("close or detached notification queried terminal coverage")
				return false
			})
			var want ssa.Value
			if name == "immediate" || name == "deferred" {
				want = function.Params[0]
			}
			if got != want {
				t.Fatalf("notification=%v, want %v", got, want)
			}
		})
	}
}
