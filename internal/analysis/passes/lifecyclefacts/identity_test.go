package lifecyclefacts

import (
	"go/types"
	"testing"

	"github.com/kojah/gohawk/internal/engine/lifecycle"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"golang.org/x/tools/go/ssa"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"golang.org/x/tools/go/analysis"
)

func TestInvocationSummaryRequiresDefiniteIdentity(t *testing.T) {
	pkg := buildLifecycleTestSSA(t, `
package lifecyclefactstest
func chosen(a, b func(), pick bool) {
	fn := a
	if pick { fn = b }
	fn()
}
func exact(a func()) { a() }
func helper(fn func()) {}
func forward(a, b func(), pick bool) {
	fn := a
	if pick { fn = b }
	helper(fn)
}
`)
	pass := &analysis.Pass{ImportObjectFact: func(types.Object, analysis.Fact) bool { return false }}
	for _, test := range []struct {
		name string
		want bool
	}{
		{"chosen", false}, {"exact", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fact := summarize(pass, pkg.Func(test.name))
			if fact.InvokedParameters().contains(0) != test.want || fact.SynchronouslyInvoked().contains(0) != test.want {
				t.Errorf("summary = %#v, want invocation of parameter 0: %t", fact, test.want)
			}
		})
	}
	fn := pkg.Func("forward")
	call := findLifecycleCall(t, fn, "helper")
	if factOwnsExactArgument(call, fn.Params[0], parameterMaskFor(0)) {
		t.Fatal("a summary for the selected argument must not prove action on either possible argument")
	}
	pass.ResultOf = map[*analysis.Analyzer]any{
		Analyzer: Summaries{call.Common().StaticCallee(): {Discharges: []Discharge{{Parameter: 0, Method: InvokeMethod}}}},
	}
	proof := NewLifecycleEvidence(pass, "test", "test/check").Prove(EvidenceRequest{
		Instruction: call, Target: fn.Params[0], SelectMask: func(fact Fact) ParameterMask { return fact.InvokedParameters() },
	})
	if proof.State != proofs.EvidenceUnknown {
		t.Errorf("ambiguous imported callback proof = %#v, want unknown", proof)
	}
}

func TestReturnedUnchangedRequiresDefiniteIdentity(t *testing.T) {
	pkg := buildLifecycleTestSSA(t, `
package lifecyclefactstest
func chosen(a, b *int, pick bool) *int {
	value := a
	if pick { value = b }
	return value
}
func exact(a *int) *int { return a }
func deferred(a *int) *int {
	defer func() { _ = *a }()
	return a
}
func replaced(a, b *int) *int {
	defer func() { _ = *a }()
	a = b
	return a
}
func namedMutated(a, b *int) (result *int) {
	defer func() { result = b }()
	return a
}
func escaped(a *int, mutate func(**int)) *int {
	mutate(&a)
	return a
}
`)
	for _, test := range []struct {
		name string
		want bool
	}{
		{"chosen", false},
		{"exact", true},
		{"deferred", true},
		{"replaced", false},
		{"namedMutated", false},
		{"escaped", false},
	} {
		fn := pkg.Func(test.name)
		if got := parameterReturnedUnchangedOnEveryReturn(fn, fn.Params[0]); got != test.want {
			t.Errorf("%s returned unchanged = %t, want %t", test.name, got, test.want)
		}
	}
}

// A completion search that runs out of budget has decided nothing, so a
// transfer check that finds no handoff must not turn it into a disproof.
func TestAbandonedCompletionStaysUndecided(t *testing.T) {
	pkg := buildLifecycleTestSSA(t, `
package lifecyclefactstest
type command struct{ done bool }
func (c *command) Wait() { c.done = true }
func helper(c *command, n int) {
	for i := 0; i < n; i++ {
		if i%2 == 0 { continue }
	}
	c.Wait()
}
func run(c *command, n int) { helper(c, n) }
`)
	pass := &analysis.Pass{ImportObjectFact: func(types.Object, analysis.Fact) bool { return false }}
	fn := pkg.Func("run")
	call := findLifecycleCall(t, fn, "helper")
	target := fn.Params[0]
	proof := NewLifecycleEvidence(pass, "test", "test/check").Prove(EvidenceRequest{
		Instruction: call, Target: target,
		Completion: &lifecycle.CompletionRequest{
			Instruction: call, Target: target, Methods: []string{"Wait"}, Budget: proofs.NewSearchBudget(1),
		},
		Transfer: &lifecycle.OwnershipTransferRequest{Instruction: call, Value: target, Modes: lifecycle.TransferStoredInGlobal},
	})
	if proof.State == proofs.EvidenceDisproven || proof.Reason != proofs.EvidenceBudgetExhausted {
		t.Errorf("abandoned completion = %#v, want an undecided budget-exhausted proof", proof)
	}
}

// Positive supplied evidence must stop before a completion walk or imported
// lookup. An abandoned local search likewise cannot become a summary disproof.
func TestEvidenceCompositionPreservesEarlyProofPayloads(t *testing.T) {
	for _, test := range []struct {
		name   string
		state  proofs.EvidenceState
		reason proofs.EvidenceReason
	}{
		{"supplied positive", proofs.EvidenceProven, proofs.EvidenceSameValue},
		{"budget cutoff", proofs.EvidenceUnknown, proofs.EvidenceBudgetExhausted},
		{"completion in cycle", proofs.EvidenceUnknown, proofs.EvidenceCompletionInCycle},
	} {
		t.Run(test.name, func(t *testing.T) {
			local := proofs.Proof{
				State: test.state, Reason: test.reason, Method: "Close", Provenance: proofs.EvidenceFromImportedFact,
			}
			request := EvidenceRequest{Local: &local, SelectMask: func(Fact) ParameterMask {
				t.Fatal("early local answer must not select an imported mask")
				return 0
			}}
			if test.state == proofs.EvidenceProven {
				// No valid instruction or allowance: evaluating this request is
				// both unnecessary and incapable of proving completion.
				request.Completion = &lifecycle.CompletionRequest{}
			}
			// No analysis pass: an imported lookup would also fail this test.
			got := (&LifecycleEvidence{}).Prove(request)
			if got != (Proof{Proof: local}) {
				t.Fatalf("proof = %+v, want exact supplied payload %+v", got, local)
			}
		})
	}
}

func TestLocalEvidenceKeepsDisproofWithoutAnotherRequest(t *testing.T) {
	local := proofs.Proof{
		State: proofs.EvidenceDisproven, Reason: proofs.EvidenceNotFound,
		Method: "Close", Provenance: proofs.EvidenceFromImportedFact,
	}
	if got := (&LifecycleEvidence{}).localProof(EvidenceRequest{Local: &local}); got != (Proof{Proof: local}) {
		t.Fatalf("proof = %+v, want exact supplied disproof %+v", got, local)
	}
}

// A positive transfer is independent evidence and can settle an abandoned
// completion. Only a failed or unknown transfer must preserve that cutoff.
func TestPositiveTransferPrecedesAbandonedCompletion(t *testing.T) {
	pkg := buildLifecycleTestSSA(t, `package lifecyclefactstest
type resource struct{}
var saved *resource
func run(value *resource) { saved = value }
`)
	function := pkg.Func("run")
	store := ssaflow.InstructionsOf[*ssa.Store](function)[0]
	local := proofs.Proof{Reason: proofs.EvidenceBudgetExhausted, Method: "Close"}
	evidence := &LifecycleEvidence{}
	got := evidence.localProof(EvidenceRequest{
		Local: &local,
		Transfer: &lifecycle.OwnershipTransferRequest{
			Instruction: store, Value: function.Params[0], Modes: lifecycle.TransferStoredInGlobal,
		},
	})
	if !got.Proven() || got.Reason != proofs.EvidenceStoredInGlobal || got.SummaryReason != reasonNone {
		t.Fatalf("proof = %+v, want exact global transfer despite completion cutoff", got)
	}
}
