package lockpolicy_test

import (
	"reflect"
	"testing"

	"github.com/kojah/gohawk/internal/analysis/analyzers/concurrency/lockorder/lockpolicy"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
)

type queries struct {
	calls                               []string
	spend, uncertain, released, returns bool
	private, held, caller               lockpolicy.Answer
	guard                               lockpolicy.Proof
	storage, heldWriter                 bool
	writer                              lockpolicy.WriterEvidence
}

func (q *queries) record(name string) { q.calls = append(q.calls, name) }

func (q *queries) Spend() bool { q.record("spend"); return q.spend }

func (q *queries) Uncertain() bool { q.record("uncertain"); return q.uncertain }

func (q *queries) Private() lockpolicy.Answer { q.record("private"); return q.private }

func (q *queries) Released() bool { q.record("released"); return q.released }

func (q *queries) Held() lockpolicy.Answer { q.record("held"); return q.held }

func (q *queries) Caller() lockpolicy.Answer { q.record("caller"); return q.caller }

func (q *queries) Returns() bool { q.record("returns"); return q.returns }

func (q *queries) Guard() lockpolicy.Proof { q.record("guard"); return q.guard }

func (q *queries) PrivateStorage() bool { q.record("storage"); return q.storage }

func (q *queries) HeldWriter() bool { q.record("held-writer"); return q.heldWriter }

func (q *queries) Writer() lockpolicy.WriterEvidence { q.record("writer"); return q.writer }

func missingQueries() *queries { return &queries{spend: true, released: true, returns: true} }

func TestMissingReleaseStopsBeforeLaterEvidence(t *testing.T) {
	all := []string{"spend", "uncertain", "private", "released", "held", "caller", "returns"}
	for _, test := range []struct {
		name   string
		change func(*queries)
		state  proofs.EvidenceState
		reason lockpolicy.Reason
		prefix int
	}{
		{"initial cutoff", func(q *queries) { q.spend = false; q.uncertain = true }, proofs.EvidenceUnknown, lockpolicy.LockStateBudgetExhausted, 1},
		{"uncertain acquisition", func(q *queries) { q.uncertain = true }, proofs.EvidenceUnknown, lockpolicy.LoadedAcquisitionGuardUnknown, 2},
		{
			"private cutoff beats witness", func(q *queries) { q.private = lockpolicy.Answer{Proven: true, Exhausted: true} },
			proofs.EvidenceUnknown, lockpolicy.LockStateBudgetExhausted, 3,
		},
		{"private completed", func(q *queries) { q.private.Proven = true }, proofs.EvidenceDisproven, lockpolicy.PrivateMutexOnly, 3},
		{"release ownership unknown", func(q *queries) { q.released = false }, proofs.EvidenceUnknown, lockpolicy.ReleaseOwnershipUnknown, 4},
		{
			"held cutoff beats contract", func(q *queries) {
				q.held = lockpolicy.Answer{Proven: true, Reason: lockpolicy.HeldForCallerProven, Exhausted: true}
			},
			proofs.EvidenceUnknown, lockpolicy.LockStateBudgetExhausted, 5,
		},
		{
			"held for caller", func(q *queries) { q.held = lockpolicy.Answer{Proven: true, Reason: lockpolicy.HeldForCallerProven} },
			proofs.EvidenceDisproven, lockpolicy.HeldForCallerProven, 5,
		},
		{
			"caller cutoff beats contract", func(q *queries) { q.caller = lockpolicy.Answer{Proven: true, Exhausted: true} },
			proofs.EvidenceUnknown, lockpolicy.LockStateBudgetExhausted, 6,
		},
		{
			"conditional caller release", func(q *queries) {
				q.caller = lockpolicy.Answer{Proven: true, Reason: lockpolicy.ConditionalCallerReleaseProven}
			},
			proofs.EvidenceDisproven, lockpolicy.ConditionalCallerReleaseProven, 6,
		},
		{"no uncovered returns", func(q *queries) { q.returns = false }, proofs.EvidenceDisproven, lockpolicy.None, 7},
		{"unreleased return", func(*queries) {}, proofs.EvidenceProven, lockpolicy.UnreleasedReturn, 7},
	} {
		t.Run(test.name, func(t *testing.T) {
			q := missingQueries()
			test.change(q)
			got := lockpolicy.MissingRelease(q)
			if got.State != test.state || got.Reason != test.reason || !reflect.DeepEqual(q.calls, all[:test.prefix]) {
				t.Fatalf("proof=%+v calls=%v", got, q.calls)
			}
		})
	}
}

func TestReadWriteDistinguishesContinueFromStop(t *testing.T) {
	all := []string{"guard", "storage", "held-writer", "writer"}
	for _, test := range []struct {
		name   string
		change func(*queries)
		state  proofs.EvidenceState
		reason lockpolicy.Reason
		stop   bool
		prefix int
	}{
		{
			"field uncertainty continues", func(q *queries) {
				q.guard = lockpolicy.Proof{State: proofs.EvidenceUnknown, Reason: lockpolicy.FieldGuardUnknown}
			},
			proofs.EvidenceUnknown, lockpolicy.FieldGuardUnknown, false, 1,
		},
		{"private destination continues", func(q *queries) { q.storage = true }, proofs.EvidenceDisproven, lockpolicy.PrivateWriteStorage, false, 2},
		{"held writer continues", func(q *queries) { q.heldWriter = true }, proofs.EvidenceUnknown, lockpolicy.ExclusiveWriterUnknown, false, 3},
		{
			"writer cutoff stops", func(q *queries) { q.writer.State = proofs.EvidenceUnknown },
			proofs.EvidenceUnknown, lockpolicy.LockStateBudgetExhausted, true, 4,
		},
		{
			"possible writer continues",
			func(q *queries) { q.writer.State = proofs.EvidenceProven },
			proofs.EvidenceUnknown,
			lockpolicy.ImportedWriterGuardUnknown,
			false,
			4,
		},
		{"unguarded write stops", func(*queries) {}, proofs.EvidenceProven, lockpolicy.ReadLockWrite, true, 4},
		{
			"known field proof alone is not a report", func(q *queries) { q.guard.State = proofs.EvidenceProven; q.storage = true },
			proofs.EvidenceDisproven, lockpolicy.PrivateWriteStorage, false, 2,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			q := &queries{guard: lockpolicy.Proof{State: proofs.EvidenceDisproven}, writer: lockpolicy.WriterEvidence{State: proofs.EvidenceDisproven}}
			test.change(q)
			got := lockpolicy.ReadWrite(q)
			if got.Proof.State != test.state || got.Proof.Reason != test.reason || got.Stop != test.stop || !reflect.DeepEqual(q.calls, all[:test.prefix]) {
				t.Fatalf("step=%+v calls=%v", got, q.calls)
			}
		})
	}
}
