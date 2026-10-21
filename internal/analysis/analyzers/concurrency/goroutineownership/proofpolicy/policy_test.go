package proofpolicy_test

import (
	"reflect"
	"testing"

	"github.com/kojah/gohawk/internal/analysis/analyzers/concurrency/goroutineownership/proofpolicy"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	ssapath "github.com/kojah/gohawk/internal/engine/ssaflow/path"
	"golang.org/x/tools/go/ssa"
)

type queries struct {
	calls     []string
	yes       map[string]bool
	lifecycle *proofpolicy.Proof
	flow      proofpolicy.Flow
	relay     proofpolicy.Relay
	census    proofpolicy.Census
	cutoff    *proofs.SearchBudget
}

func (q *queries) record(name string) bool { q.calls = append(q.calls, name); return q.yes[name] }
func (q *queries) Lifecycle() (proofpolicy.Proof, bool) {
	q.record("lifecycle")
	if q.lifecycle != nil {
		return *q.lifecycle, true
	}
	return proofpolicy.Proof{}, false
}
func (q *queries) Relay() proofpolicy.Relay { q.record("relay"); return q.relay }
func (q *queries) RelayCutoff(budget *proofs.SearchBudget) proofpolicy.Proof {
	q.record("relay-cutoff")
	q.cutoff = budget
	return proofpolicy.Proof{Outcome: proofpolicy.Unknown, Reason: proofpolicy.RelayDependencyBudgetExhausted}
}

func (q *queries) CompletionHandle() (proofpolicy.Proof, bool) {
	q.record("handle")
	return proofpolicy.Proof{}, false
}

func (q *queries) CallerLifetime() (proofpolicy.Proof, bool) {
	q.record("caller")
	return proofpolicy.Proof{}, false
}
func (q *queries) Synctest() bool { return q.record("synctest") }
func (q *queries) Dominating() (proofpolicy.Proof, bool) {
	q.record("dominating")
	return proofpolicy.Proof{}, false
}
func (q *queries) Workers() bool          { return q.record("workers") }
func (q *queries) Flow() proofpolicy.Flow { q.record("flow"); return q.flow }
func (q *queries) Honored(flow proofpolicy.Flow) bool {
	q.record("honored")
	return flow.Outcome == ssapath.ObligationHonored
}
func (q *queries) Guarded() bool { return q.record("guarded") }
func (q *queries) Uncertain(flow proofpolicy.Flow) bool {
	q.record("uncertain")
	return flow.Outcome == ssapath.ObligationUncertain
}
func (q *queries) FlagGuarded() bool          { return q.record("flag") }
func (q *queries) Counted() bool              { return q.record("counted") }
func (q *queries) SharedStorage() bool        { return q.record("shared") }
func (q *queries) Buffered() bool             { return q.record("buffered") }
func (q *queries) Census() proofpolicy.Census { q.record("census"); return q.census }
func (q *queries) CensusCutoff(budget *proofs.SearchBudget) proofpolicy.Proof {
	q.record("census-cutoff")
	q.cutoff = budget
	return proofpolicy.Proof{Outcome: proofpolicy.Unknown, Reason: proofpolicy.SignalCensusUnavailable}
}
func (q *queries) ProgramEntry() bool { return q.record("entry") }

func TestDiscoveryAvailabilityPrecedesEveryQuery(t *testing.T) {
	for _, test := range []struct {
		discovery proofpolicy.Discovery
		reason    proofpolicy.Reason
	}{
		{proofpolicy.Discovery{Exhausted: true, EarlyDone: true}, proofpolicy.DiscoveryBudgetExhausted},
		{proofpolicy.Discovery{EarlyDone: true}, proofpolicy.DoneBeforeCompletion},
		{proofpolicy.Discovery{}, proofpolicy.NoObligation},
	} {
		host := &queries{}
		got := proofpolicy.Prove(test.discovery, host)
		if got.Outcome != proofpolicy.Unknown || got.Reason != test.reason || len(host.calls) != 0 {
			t.Fatalf("%+v: proof=%+v calls=%v", test.discovery, got, host.calls)
		}
	}
}

func TestProofQueriesShortCircuitInEvidenceOrder(t *testing.T) {
	prefix := []string{"lifecycle", "dominating", "workers", "flow", "honored", "guarded"}
	for _, test := range []struct {
		name    string
		flow    ssapath.ObligationOutcome
		guarded bool
		outcome proofpolicy.Outcome
		reason  proofpolicy.Reason
		calls   []string
	}{
		{"honored", ssapath.ObligationHonored, false, proofpolicy.Honored, proofpolicy.JoinProven, prefix[:5]},
		{"guarded-before-opacity", ssapath.ObligationUncertain, true, proofpolicy.Honored, proofpolicy.GuardedLocalJoin, prefix},
		{"opaque", ssapath.ObligationUncertain, false, proofpolicy.Unknown, proofpolicy.OpaqueTransfer, append(append([]string(nil), prefix...), "uncertain")},
	} {
		t.Run(test.name, func(t *testing.T) {
			host := &queries{flow: proofpolicy.Flow{Outcome: test.flow}, yes: map[string]bool{"guarded": test.guarded}}
			got := proofpolicy.Prove(proofpolicy.Discovery{Obligation: true}, host)
			if got.Outcome != test.outcome || got.Reason != test.reason || !reflect.DeepEqual(host.calls, test.calls) {
				t.Fatalf("proof=%+v calls=%v", got, host.calls)
			}
		})
	}
}

func TestUnknownLifecycleAnswerStopsFurtherStrategies(t *testing.T) {
	proof := proofpolicy.Proof{Outcome: proofpolicy.Unknown, Reason: proofpolicy.StopLifecycle}
	host := &queries{lifecycle: &proof}
	if got := proofpolicy.Prove(proofpolicy.Discovery{Obligation: true}, host); got != proof || !reflect.DeepEqual(host.calls, []string{"lifecycle"}) {
		t.Fatalf("proof=%+v calls=%v", got, host.calls)
	}
}

func TestCensusCutoffRetainsItsChildAllowance(t *testing.T) {
	pool := proofs.NewSearchBudget(proofs.QueryBudget)
	child := pool.Within(0)
	child.Spend()
	host := &queries{census: proofpolicy.Census{Kind: proofpolicy.CensusInterrupted, Budget: child}, yes: map[string]bool{"entry": true}}
	got := proofpolicy.Fallback(host, &ssa.Return{})
	want := []string{"flag", "counted", "shared", "buffered", "census", "census-cutoff"}
	if got.Outcome != proofpolicy.Unknown || got.Reason != proofpolicy.SignalCensusUnavailable || host.cutoff != child || pool.Exhausted() ||
		!reflect.DeepEqual(host.calls, want) {
		t.Fatalf("proof=%+v calls=%v cutoff=%p", got, host.calls, host.cutoff)
	}
}

func TestRelayCutoffPrecedesOwnershipQueries(t *testing.T) {
	budget := proofs.NewSearchBudget(0)
	budget.Spend()
	host := &queries{relay: proofpolicy.Relay{Kind: proofpolicy.RelayInterrupted, Budget: budget}}
	got, decided := proofpolicy.Lifecycle(host)
	if !decided || got.Reason != proofpolicy.RelayDependencyBudgetExhausted || host.cutoff != budget ||
		!reflect.DeepEqual(host.calls, []string{"relay", "relay-cutoff"}) {
		t.Fatalf("proof=%+v decided=%v calls=%v", got, decided, host.calls)
	}
}

func (q *queries) Fallback(witness *ssa.Return) proofpolicy.Proof {
	return proofpolicy.Fallback(q, witness)
}
