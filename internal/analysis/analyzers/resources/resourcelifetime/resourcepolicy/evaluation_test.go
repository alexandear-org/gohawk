package resourcepolicy_test

import (
	"reflect"
	"testing"

	"github.com/kojah/gohawk/internal/analysis/analyzers/resources/resourcelifetime/resourcepolicy"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"golang.org/x/tools/go/ssa"
)

type evaluationState struct {
	calls                                  []string
	values                                 []ssa.Value
	memory                                 bool
	paired                                 resourcepolicy.PairedError
	boundary                               resourcepolicy.Reason
	canceled, asserted, optional, prepared resourcepolicy.Evidence
	flow                                   resourcepolicy.Decision
	database, pool                         bool
	exit                                   resourcepolicy.ExitEvidence
}

type phaseQueries struct {
	state *evaluationState
	phase string
}

func (q phaseQueries) record(name string) { q.state.calls = append(q.state.calls, q.phase+"."+name) }

func (q phaseQueries) MemoryWriter() bool { q.record("memory"); return q.state.memory }

func (q phaseQueries) Begin() resourcepolicy.AcquisitionQueries {
	q.record("begin")
	return phaseQueries{state: q.state, phase: "acquisition"}
}

func (q phaseQueries) Error() resourcepolicy.PairedError { q.record("error"); return q.state.paired }

func (q phaseQueries) Boundary() resourcepolicy.Reason { q.record("boundary"); return q.state.boundary }

func (q phaseQueries) Canceled() resourcepolicy.Evidence {
	q.record("canceled")
	return q.state.canceled
}

func (q phaseQueries) Asserted(v ssa.Value) resourcepolicy.Evidence {
	q.record("asserted")
	q.state.values = append(q.state.values, v)
	return q.state.asserted
}

func (q phaseQueries) Optional(v ssa.Value) resourcepolicy.Correlation {
	q.record("optional")
	q.state.values = append(q.state.values, v)
	p := q.state.optional
	return resourcepolicy.Correlation{State: p.State, Reason: p.Reason, Initialize: phaseQueries{state: q.state, phase: "initialization"}}
}

func (q phaseQueries) Prepare() resourcepolicy.Prepared {
	q.record("prepare")
	p := q.state.prepared
	return resourcepolicy.Prepared{State: p.State, Reason: p.Reason, Queries: phaseQueries{state: q.state, phase: "flow"}}
}

func (q phaseQueries) Flow(v ssa.Value) resourcepolicy.Decision {
	q.record("flow")
	q.state.values = append(q.state.values, v)
	return q.state.flow
}

func (q phaseQueries) DatabaseCleanup() bool { q.record("database"); return q.state.database }

func (q phaseQueries) PoolExhausted() bool { q.record("pool"); return q.state.pool }

func (q phaseQueries) Exit() resourcepolicy.ExitEvidence { q.record("exit"); return q.state.exit }

func newEvaluationState() *evaluationState {
	absent := resourcepolicy.Evidence{State: proofs.EvidenceDisproven}
	return &evaluationState{
		paired:   resourcepolicy.PairedError{State: proofs.EvidenceDisproven, Value: &ssa.Parameter{}},
		canceled: absent, asserted: absent, optional: absent,
		prepared: resourcepolicy.Evidence{State: proofs.EvidenceProven},
		flow:     resourcepolicy.Decision{State: proofs.EvidenceProven, Reason: resourcepolicy.UnownedReturn, Leak: &ssa.Return{}},
	}
}

func TestEvaluationStopsBeforeAnyLaterPhaseAtCutoff(t *testing.T) {
	all := []string{
		"root.memory", "root.begin", "acquisition.error", "acquisition.boundary", "acquisition.canceled",
		"acquisition.asserted", "acquisition.optional", "initialization.prepare", "flow.flow", "flow.database", "flow.pool", "flow.exit",
	}
	cutoff := resourcepolicy.Evidence{State: proofs.EvidenceUnknown, Reason: resourcepolicy.BudgetExhausted}
	positive := resourcepolicy.Evidence{State: proofs.EvidenceProven}
	for _, test := range []struct {
		name   string
		change func(*evaluationState)
		state  proofs.EvidenceState
		reason resourcepolicy.Reason
		prefix int
	}{
		{"memory exclusion", func(s *evaluationState) { s.memory = true }, proofs.EvidenceDisproven, resourcepolicy.MemoryWriter, 1},
		{
			"error cutoff", func(s *evaluationState) { s.paired.State = cutoff.State; s.paired.Reason = cutoff.Reason },
			proofs.EvidenceUnknown, resourcepolicy.BudgetExhausted, 3,
		},
		{
			"HTTP boundary", func(s *evaluationState) { s.boundary = resourcepolicy.HeadClientNotUnconfigured },
			proofs.EvidenceUnknown, resourcepolicy.HeadClientNotUnconfigured, 4,
		},
		{"cancellation cutoff", func(s *evaluationState) { s.canceled = cutoff }, proofs.EvidenceUnknown, resourcepolicy.BudgetExhausted, 5},
		{"canceled acquisition", func(s *evaluationState) { s.canceled = positive }, proofs.EvidenceDisproven, resourcepolicy.CanceledAcquisition, 5},
		{"assertion cutoff", func(s *evaluationState) { s.asserted = cutoff }, proofs.EvidenceUnknown, resourcepolicy.BudgetExhausted, 6},
		{"asserted failure", func(s *evaluationState) { s.asserted = positive }, proofs.EvidenceDisproven, resourcepolicy.ReleaseProven, 6},
		{"optional cutoff", func(s *evaluationState) { s.optional = cutoff }, proofs.EvidenceUnknown, resourcepolicy.BudgetExhausted, 7},
		{"preparation cutoff", func(s *evaluationState) { s.prepared = cutoff }, proofs.EvidenceUnknown, resourcepolicy.BudgetExhausted, 8},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := newEvaluationState()
			test.change(s)
			got := resourcepolicy.Evaluate(phaseQueries{state: s, phase: "root"})
			if got.State != test.state || got.Reason != test.reason || got.Leak != nil || !reflect.DeepEqual(s.calls, all[:test.prefix]) {
				t.Fatalf("decision=%+v calls=%v", got, s.calls)
			}
			for _, v := range s.values {
				if v != s.paired.Value {
					t.Fatal("paired error value changed between phases")
				}
			}
		})
	}
}

func TestFinalExclusionsPreservePrecedenceAndWitnesses(t *testing.T) {
	all := []string{"flow.flow", "flow.database", "flow.pool", "flow.exit"}
	for _, test := range []struct {
		name        string
		change      func(*evaluationState)
		reason      resourcepolicy.Reason
		state       proofs.EvidenceState
		prefix      int
		keepWitness bool
	}{
		{
			"unknown flow stops", func(s *evaluationState) { s.flow.State = proofs.EvidenceUnknown },
			resourcepolicy.UnownedReturn, proofs.EvidenceUnknown, 1, true,
		},
		{
			"accepted flow stops", func(s *evaluationState) { s.flow.State = proofs.EvidenceDisproven },
			resourcepolicy.UnownedReturn, proofs.EvidenceDisproven, 1, true,
		},
		{
			"database before pool cutoff", func(s *evaluationState) { s.database = true; s.pool = true },
			resourcepolicy.ParentCleanup, proofs.EvidenceDisproven, 2, false,
		},
		{
			"pool cutoff before exit", func(s *evaluationState) { s.pool = true; s.exit.Reclaimed = true },
			resourcepolicy.BudgetExhausted, proofs.EvidenceUnknown, 3, false,
		},
		{
			"positive exit before child cutoff", func(s *evaluationState) { s.exit = resourcepolicy.ExitEvidence{Reclaimed: true, Exhausted: true} },
			resourcepolicy.ProcessExitReclaims, proofs.EvidenceDisproven, 4, false,
		},
		{
			"exit cutoff discards witness", func(s *evaluationState) { s.exit.Exhausted = true },
			resourcepolicy.BudgetExhausted, proofs.EvidenceUnknown, 4, false,
		},
		{"uncovered return survives", func(*evaluationState) {}, resourcepolicy.UnownedReturn, proofs.EvidenceProven, 4, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := newEvaluationState()
			test.change(s)
			got := resourcepolicy.Finalize(phaseQueries{state: s, phase: "flow"}, s.paired.Value)
			var witness *ssa.Return
			if test.keepWitness {
				witness = s.flow.Leak
			}
			if got.State != test.state || got.Reason != test.reason || got.Leak != witness || !reflect.DeepEqual(s.calls, all[:test.prefix]) {
				t.Fatalf("decision=%+v calls=%v, want witness=%p", got, s.calls, witness)
			}
		})
	}
}

func TestEvaluationForwardsTheCompletedPhaseContexts(t *testing.T) {
	s := newEvaluationState()
	got := resourcepolicy.Evaluate(phaseQueries{state: s, phase: "root"})
	want := []string{
		"root.memory", "root.begin", "acquisition.error", "acquisition.boundary", "acquisition.canceled",
		"acquisition.asserted", "acquisition.optional", "initialization.prepare", "flow.flow", "flow.database", "flow.pool", "flow.exit",
	}
	if got != s.flow || !reflect.DeepEqual(s.calls, want) {
		t.Fatalf("decision=%+v calls=%v", got, s.calls)
	}
	if len(s.values) != 3 {
		t.Fatalf("paired value forwarded %d times", len(s.values))
	}
	for _, v := range s.values {
		if v != s.paired.Value {
			t.Fatal("completed negative paired-error evidence lost its payload")
		}
	}
}

type setupQueries struct {
	calls                    []string
	deferred, guarded, prior resourcepolicy.Evidence
	completeOwners           bool
}

func (q *setupQueries) Deferred() resourcepolicy.Evidence {
	q.calls = append(q.calls, "deferred")
	return q.deferred
}

func (q *setupQueries) Owners() bool { q.calls = append(q.calls, "owners"); return q.completeOwners }

func (q *setupQueries) Guarded() resourcepolicy.Evidence {
	q.calls = append(q.calls, "collection/guarded")
	return q.guarded
}

func (q *setupQueries) Prior() resourcepolicy.Evidence {
	q.calls = append(q.calls, "prior")
	return q.prior
}

func TestPreparationRequiresCompleteClassifierInputs(t *testing.T) {
	absent := resourcepolicy.Evidence{State: proofs.EvidenceDisproven}
	positive := resourcepolicy.Evidence{State: proofs.EvidenceProven}
	cutoff := resourcepolicy.Evidence{State: proofs.EvidenceUnknown, Reason: resourcepolicy.EvidenceUnavailable, Provenance: proofs.EvidenceFromLocalSSA}
	all := []string{"deferred", "owners", "collection/guarded", "prior"}
	for _, test := range []struct {
		name   string
		change func(*setupQueries)
		reason resourcepolicy.Reason
		state  proofs.EvidenceState
		prefix int
	}{
		{"deferred cutoff", func(q *setupQueries) { q.deferred = cutoff }, resourcepolicy.EvidenceUnavailable, proofs.EvidenceUnknown, 1},
		{"possible deferred release", func(q *setupQueries) { q.deferred = positive }, resourcepolicy.PriorDeferMayRelease, proofs.EvidenceUnknown, 1},
		{"owner census incomplete", func(q *setupQueries) { q.completeOwners = false }, resourcepolicy.BudgetExhausted, proofs.EvidenceUnknown, 2},
		{"guarded cutoff", func(q *setupQueries) { q.guarded = cutoff }, resourcepolicy.EvidenceUnavailable, proofs.EvidenceUnknown, 3},
		{"prior cutoff", func(q *setupQueries) { q.prior = cutoff }, resourcepolicy.EvidenceUnavailable, proofs.EvidenceUnknown, 4},
		{"possible prior cleanup", func(q *setupQueries) { q.prior = positive }, resourcepolicy.OpaqueConsumption, proofs.EvidenceUnknown, 4},
		{"positive guarded evidence continues", func(q *setupQueries) { q.guarded = positive }, resourcepolicy.None, proofs.EvidenceProven, 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			q := &setupQueries{deferred: absent, guarded: absent, prior: absent, completeOwners: true}
			test.change(q)
			got := resourcepolicy.PrepareFlow(q)
			if got.State != test.state || got.Reason != test.reason || got.Provenance != 0 || !reflect.DeepEqual(q.calls, all[:test.prefix]) {
				t.Fatalf("proof=%+v calls=%v", got, q.calls)
			}
		})
	}
}
