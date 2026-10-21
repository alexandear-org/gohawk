package goroutineownership

import (
	"testing"

	"github.com/kojah/gohawk/internal/analysis/passes/concurrencyfacts"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ssa"
)

func TestInstructionLabelBoundaries(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "labels", `package labels
 import "sync"
 type owner struct{done chan int}
 var saved chan int
 func receive(done chan int){<-done}
 func unrelated(done, other chan int){<-other}
 func merged(done, other chan int, choose bool){c:=other; if choose{c=done}; <-c}
 func aggregate(p *owner){<-p.done}
 func returned(done chan int)chan int{return done}
 func returnedChoice(done, other chan int, choose bool)chan int{c:=other; if choose{c=done}; return c}
 func stored(done chan int){saved=done}
 func sent(done chan int, destination chan (chan int)){destination<-done}
 func mapped(done chan int, destination map[int]chan int){destination[0]=done}
 func captured(done chan int)func(){return func(){<-done}}
 func selected(done, other chan int){select{case <-done: done<-1; case <-other:}}
 func appended(done chan int, dst []chan int)[]chan int{return append(dst,done)}
 func appendOther(done,other chan int,dst []chan int)[]chan int{return append(dst,other)}
 func capacity(done chan int)int{return cap(done)}
 func closed(done chan int){close(done)}
 func launchedClose(done chan int){go close(done)}
 func possibleWait(group,other *sync.WaitGroup,choose bool){selected:=other; if choose{selected=group}; selected.Wait()}
 func wait(group *sync.WaitGroup){group.Wait()}
 func deferred(group *sync.WaitGroup){defer group.Wait()}
 func launched(group *sync.WaitGroup){go group.Wait()}
 `)
	for _, test := range []struct {
		name        string
		instruction func(*ssa.Function) ssa.Instruction
		action      ownershipAction
		reason      goroutineOwnershipReason
		group       bool
	}{
		{"receive", lastLabelInstruction[*ssa.UnOp], actionJoin, reasonLabelSignalReceived, false},
		{"unrelated", lastLabelInstruction[*ssa.UnOp], actionNone, reasonNone, false},
		{"merged", lastLabelInstruction[*ssa.UnOp], actionUnknown, reasonLabelPossibleSignalReceive, false},
		{"aggregate", lastLabelInstruction[*ssa.UnOp], actionUnknown, reasonLabelPossibleSignalReceive, false},
		{"returned", lastLabelInstruction[*ssa.Return], actionTransfer, reasonLabelReturnedTracked, false},
		{"returnedChoice", lastLabelInstruction[*ssa.Return], actionUnknown, reasonLabelReturnedContainment, false},
		{"stored", lastLabelInstruction[*ssa.Store], actionTransfer, reasonLabelStoredOutside, false},
		{"sent", lastLabelInstruction[*ssa.Send], actionUnknown, reasonLabelSent, false},
		{"mapped", lastLabelInstruction[*ssa.MapUpdate], actionUnknown, reasonLabelStoredInMap, false},
		{"captured", lastLabelInstruction[*ssa.MakeClosure], actionNone, reasonNone, false},
		// The selected receive settles its own arm before the send consumes it.
		{"selected", lastLabelInstruction[*ssa.Send], actionJoin, reasonLabelSignalReceived, false},
		{"appended", lastLabelInstruction[*ssa.Call], actionUnknown, reasonLabelAppended, false},
		{"appendOther", lastLabelInstruction[*ssa.Call], actionNone, reasonNone, false},
		{"capacity", lastLabelInstruction[*ssa.Call], actionNone, reasonNone, false},
		{"closed", lastLabelInstruction[*ssa.Call], actionNone, reasonNone, false},
		{"launchedClose", lastLabelInstruction[*ssa.Go], actionUnknown, reasonLabelLaunchedHelper, false},
		{"possibleWait", lastLabelInstruction[*ssa.Call], actionUnknown, reasonLabelPossibleJoin, true},
		{"wait", lastLabelInstruction[*ssa.Call], actionJoin, reasonLabelDirectJoin, true},
		{"deferred", lastLabelInstruction[*ssa.Defer], actionJoin, reasonLabelDirectJoin, true},
		{"launched", lastLabelInstruction[*ssa.Go], actionUnknown, reasonLabelLaunchedHelper, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := pkg.Func(test.name)
			value := function.Params[0]
			kind := trackedSignal
			if test.group {
				kind = trackedGroup
			}
			analysis := &spawnAnalysis{
				function: function, tracked: []trackedValue{{value: value, kind: kind}},
				actions: make(map[ssa.Instruction]ownershipAction), pool: proofs.NewSearchBudget(10000),
			}
			if test.group {
				analysis.groups = []ssa.Value{value}
			} else {
				analysis.signals = []ssa.Value{value}
			}
			instruction := test.instruction(function)
			action, reason := analysis.classify(instruction)
			if action != test.action || reason != test.reason {
				t.Fatalf("%s: got %v/%v, want %v/%v", instruction, action, reason, test.action, test.reason)
			}
			if got := analysis.action(instruction); got != test.action {
				t.Fatalf("cached classification = %v, want %v", got, test.action)
			}
			remaining := analysis.pool.Remaining()
			if got := analysis.action(instruction); got != test.action || analysis.pool.Remaining() != remaining {
				t.Fatalf("cached action repeated evidence queries: action=%v remaining=%d, want %d", got, analysis.pool.Remaining(), remaining)
			}
		})
	}
}

func lastLabelInstruction[T ssa.Instruction](function *ssa.Function) ssa.Instruction {
	instructions := ssaflow.InstructionsOf[T](function)
	return instructions[len(instructions)-1]
}

func TestRecognizedCallNoneStopsContractFallback(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "callcontracts", `package callcontracts
 import("sync"; "testing")
 func subject(t *testing.T){var group sync.WaitGroup; go group.Done(); group.Add(1); t.Cleanup(func(){})}
 `)
	function := pkg.Func("subject")
	calls := ssaflow.InstructionsOf[*ssa.Call](function)
	target := calls[0].Common().Args[0]
	pass := &analysis.Pass{Fset: pkg.Prog.Fset, Pkg: pkg.Pkg, ResultOf: map[*analysis.Analyzer]any{concurrencyfacts.Analyzer: concurrencyfacts.NewEngine()}}
	for index, want := range []goroutineOwnershipReason{reasonNone, reasonLabelTestingCleanup} {
		candidate := &spawnAnalysis{
			pass: pass, function: function, spawn: ssaflow.InstructionsOf[*ssa.Go](function)[0],
			groups: []ssa.Value{target}, tracked: []trackedValue{{value: target, kind: trackedGroup}},
			pool: proofs.NewSearchBudget(spawnPoolBudget),
		}
		call := calls[index]
		if prefix, decided := recognizeSpawnCall(candidate, call, call.Common()); decided {
			t.Fatalf("contract was consumed by call prefix: %+v", prefix)
		}
		label, decided := recognizeSpawnCallContract(candidate, call, call.Common())
		if !decided || label.action != actionNone || label.reason != want {
			t.Fatalf("call %s: recognized=%v label=%+v, want terminal none/%v", call, decided, label, want)
		}
	}
	empty := &spawnAnalysis{}
	if action, reason := empty.callAction(nil, nil); action != actionNone || reason != reasonNone || empty.pool != nil {
		t.Fatalf("missing call issued a query: %v/%v pool=%v", action, reason, empty.pool)
	}
}
