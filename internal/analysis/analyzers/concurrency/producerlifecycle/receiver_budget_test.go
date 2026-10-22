package producerlifecycle

import (
	"strings"
	"testing"

	"github.com/kojah/gohawk/internal/analysis/passes/concurrencyfacts"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/ssa"
)

func TestHelperReceiverCutoffKeepsUnknown(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "receiverbudget", `package receiverbudget
func receive(ch chan int) { <-ch }
func subject(ch chan int) { receive(ch) }
func receiveTwice(ch chan int) { <-ch; <-ch }
func subjectTwice(ch chan int) { receiveTwice(ch) }
func worker(ch chan int, opaque func()) { opaque(); ch <- 1 }
func launched(ch chan int, opaque func()) { go worker(ch, opaque) }
`)
	for name, count := range map[string]int{"subject": 1, "subjectTwice": 2, "launched": 0} {
		fn := pkg.Func(name)
		var dump strings.Builder
		if _, err := fn.WriteTo(&dump); err != nil {
			t.Fatal(err)
		}
		t.Log(dump.String())
		call := fn.Blocks[0].Instrs[0].(ssa.CallInstruction)
		checkHelperReceiverBudgets(t, call, fn.Params[0], count)
	}
}

func checkHelperReceiverBudgets(t *testing.T, call ssa.CallInstruction, channel ssa.Value, count int) {
	t.Helper()
	for _, warm := range []bool{false, true} {
		complete := false
		for limit := range 256 {
			engine := concurrencyfacts.NewEngine()
			if warm {
				summary := engine.AtCall(call, nil)
				if summary.Complete() != (count > 0) {
					t.Fatal("unexpected warm summary availability")
				}
			}
			budget := proofs.NewSearchBudget(limit)
			proof := helperReceives(call, channel, nil, engine, budget)
			if budget.Exhausted() {
				if !proof.unknown || proof.count != 0 {
					t.Fatalf("warm=%v cutoff=%d supplied a count: %+v", warm, limit, proof)
				}
				if proof.reason != reasonReceiverBudgetExhausted {
					t.Errorf("warm=%v cutoff=%d reason=%s", warm, limit, proof.reason.String())
				}
				continue
			}
			if proof.unknown || proof.count != count {
				t.Fatalf("warm=%v complete=%d: %+v", warm, limit, proof)
			}
			complete = true
			break
		}
		if !complete {
			t.Fatalf("warm=%v never completed", warm)
		}
	}
}

func TestHelperReceiverExclusionsDoNotQuerySummaries(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "excluded", `package excluded
func worker(ch chan int) { ch <- 1 }
func subject(ch chan int) { go worker(ch); close(ch) }
`)
	function := pkg.Func("subject")
	origin := function.Blocks[0].Instrs[0].(*ssa.Go)
	builtin := function.Blocks[0].Instrs[1].(*ssa.Call)
	for _, test := range []struct {
		name   string
		call   ssa.CallInstruction
		reason producerReason
	}{
		{"origin", origin, reasonProducerLaunch},
		{"builtin", builtin, reasonBuiltinNotReceive},
	} {
		t.Run(test.name, func(t *testing.T) {
			budget := proofs.NewSearchBudget(0)
			budget.Spend()
			// A nil engine pins the early exclusion: no summary lookup may
			// run even when an existing allowance is already exhausted.
			got := helperReceives(test.call, function.Params[0], origin, nil, budget)
			if got != (receiveProof{reason: test.reason}) {
				t.Fatalf("early exclusion = %+v, want reason %v", got, test.reason)
			}
		})
	}
}

func TestOpaqueHelperKeepsFalseUncertaintyPayload(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "opaque", `package opaque
func external()
func subject(ch chan int) { external() }
`)
	function := pkg.Func("subject")
	call := function.Blocks[0].Instrs[0].(*ssa.Call)
	got := helperReceives(call, function.Params[0], nil, concurrencyfacts.NewEngine(), nil)
	// The summary is unavailable but no input carries this channel. Keep
	// that false payload and its explanation rather than invent a receiver.
	if got != (receiveProof{reason: reasonReceiverHelperUnknown}) {
		t.Fatalf("unrelated opaque helper = %+v", got)
	}
}

func TestAbandonedProducerEarlyEvidenceSkipsReceiveQueries(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "precedence", `package precedence
func worker(ch chan int) { ch <- 1 }
func receive(ch chan int) { <-ch }
func subject() { ch := make(chan int); go worker(ch); receive(ch) }
func noReturn() { ch := make(chan int); go worker(ch); select {} }
`)
	for _, test := range []struct {
		name   string
		reason producerReason
	}{
		{"subject", reasonProducerCountUnknown},
		{"noReturn", reasonReceiverDoesNotReturn},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := pkg.Func(test.name)
			var send producerSend
			for _, instruction := range function.Blocks[0].Instrs {
				if spawn, ok := instruction.(*ssa.Go); ok {
					send = producerSend{spawn: spawn, channel: spawn.Call.Args[0], repeated: true}
				}
			}
			if send.spawn == nil {
				t.Fatal("fixture has no producer launch")
			}
			got := abandonedProducerSend(function, send, []producerSend{send}, nil)
			if got != (producerProof{Reason: test.reason}) {
				t.Fatalf("early producer evidence = %+v, want reason %v", got, test.reason)
			}
		})
	}
}
