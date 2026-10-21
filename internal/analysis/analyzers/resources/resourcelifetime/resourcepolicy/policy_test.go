package resourcepolicy_test

import (
	"reflect"
	"testing"

	"github.com/kojah/gohawk/internal/analysis/analyzers/resources/resourcelifetime/resourcepolicy"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/ssa"
)

type queries struct {
	calls     []string
	replies   map[string]resourcepolicy.Evidence
	unmodeled bool
	payload   bool
}

func (q *queries) evidence(name string) resourcepolicy.Evidence {
	q.calls = append(q.calls, name)
	if reply, ok := q.replies[name]; ok {
		return reply
	}
	return resourcepolicy.Evidence{State: proofs.EvidenceDisproven, Reason: resourcepolicy.Untouched}
}
func (q *queries) Returned(*ssa.Return) resourcepolicy.Evidence { return q.evidence("return") }
func (q *queries) Storage(*ssa.Store) resourcepolicy.Storage {
	p := q.evidence("storage")
	return resourcepolicy.Storage{State: p.State, Reason: p.Reason}
}
func (q *queries) Wrapper(*ssa.Store) resourcepolicy.Evidence { return q.evidence("wrapper") }
func (q *queries) Handoff(ssa.Value, ssa.Instruction) resourcepolicy.Evidence {
	return q.evidence("handoff")
}

func (q *queries) Payload(_ ssa.Value, reason resourcepolicy.Reason) (resourcepolicy.Reason, bool) {
	q.calls = append(q.calls, "payload")
	return reason, q.payload
}

func (q *queries) Select(*ssa.Select) (resourcepolicy.Reason, bool) {
	q.calls = append(q.calls, "select")
	return resourcepolicy.None, false
}

func (q *queries) Call(i ssa.Instruction, c *ssa.CallCommon) (resourcepolicy.Reason, bool) {
	return resourcepolicy.Call(q, i, c)
}
func (q *queries) Carried(*ssa.CallCommon) resourcepolicy.Evidence { return q.evidence("carried") }
func (q *queries) Callback(ssa.Instruction, *ssa.CallCommon) resourcepolicy.Evidence {
	return q.evidence("callback")
}

func (q *queries) Exposure(ssa.Instruction, *ssa.CallCommon) resourcepolicy.Evidence {
	return q.evidence("exposure")
}

func (q *queries) Closure(ssa.Instruction, *ssa.MakeClosure, bool) (resourcepolicy.Reason, bool) {
	q.calls = append(q.calls, "closure")
	return resourcepolicy.CapturedByRetainingLiteral, false
}

func (q *queries) Function(i ssa.Instruction, c *ssa.CallCommon, b bool) (resourcepolicy.Reason, bool) {
	q.calls = append(q.calls, "function")
	return resourcepolicy.Function(q, i, c, b)
}

func (q *queries) Chain(ssa.Instruction, *ssa.CallCommon) resourcepolicy.Evidence {
	return q.evidence("chain")
}

func (q *queries) Escape(ssa.Instruction, *ssa.CallCommon) resourcepolicy.Evidence {
	return q.evidence("escape")
}

func (q *queries) Aggregate(*ssa.CallCommon) resourcepolicy.Evidence { return q.evidence("aggregate") }

func (q *queries) Publication(ssa.Instruction) resourcepolicy.Evidence {
	return q.evidence("publication")
}

func (q *queries) Unmodeled(ssa.Instruction, *ssa.CallCommon) bool {
	q.calls = append(q.calls, "unmodeled")
	return q.unmodeled
}

func TestCallQueriesStopAtTheFirstOpaqueBoundary(t *testing.T) {
	proven := resourcepolicy.Evidence{State: proofs.EvidenceProven}
	cutoff := resourcepolicy.Evidence{State: proofs.EvidenceUnknown, Reason: resourcepolicy.BudgetExhausted}
	for _, test := range []struct {
		name    string
		replies map[string]resourcepolicy.Evidence
		reason  resourcepolicy.Reason
		opaque  bool
		calls   []string
	}{
		{"carried cutoff", map[string]resourcepolicy.Evidence{"carried": cutoff}, resourcepolicy.BudgetExhausted, true, []string{"carried"}},
		{
			"uncarried callback cutoff",
			map[string]resourcepolicy.Evidence{"callback": cutoff},
			resourcepolicy.BudgetExhausted,
			true,
			[]string{"carried", "callback"},
		},
		{
			"uncarried chain cutoff",
			map[string]resourcepolicy.Evidence{"chain": cutoff},
			resourcepolicy.BudgetExhausted,
			true,
			[]string{"carried", "callback", "function", "chain"},
		},
		{"uncarried call", nil, resourcepolicy.None, false, []string{"carried", "callback", "function", "chain"}},
		{
			"exposure cutoff",
			map[string]resourcepolicy.Evidence{"carried": proven, "exposure": cutoff},
			resourcepolicy.BudgetExhausted,
			true,
			[]string{"carried", "callback", "exposure"},
		},
		{
			"carried inspected call",
			map[string]resourcepolicy.Evidence{"carried": proven},
			resourcepolicy.UnsummarizedCallee,
			false,
			[]string{"carried", "callback", "exposure", "function", "chain", "escape", "aggregate", "unmodeled"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			q := &queries{replies: test.replies}
			reason, opaque := resourcepolicy.Call(q, &ssa.Call{}, &ssa.CallCommon{Value: &ssa.Function{}})
			if reason != test.reason || opaque != test.opaque || !reflect.DeepEqual(q.calls, test.calls) {
				t.Fatalf("reason=%v opaque=%v calls=%v", reason, opaque, q.calls)
			}
		})
	}
}

func TestAggregatePublicationRunsOnlyAfterPositiveContainment(t *testing.T) {
	for _, test := range []struct {
		state       proofs.EvidenceState
		publication proofs.EvidenceState
		reason      resourcepolicy.Reason
		opaque      bool
		calls       []string
	}{
		{proofs.EvidenceUnknown, proofs.EvidenceProven, resourcepolicy.BudgetExhausted, true, []string{"chain", "escape", "aggregate"}},
		{proofs.EvidenceDisproven, proofs.EvidenceUnknown, resourcepolicy.UnsummarizedCallee, false, []string{"chain", "escape", "aggregate", "unmodeled"}},
		{proofs.EvidenceProven, proofs.EvidenceUnknown, resourcepolicy.BudgetExhausted, true, []string{"chain", "escape", "aggregate", "publication"}},
		{
			proofs.EvidenceProven, proofs.EvidenceProven, resourcepolicy.NestedInTransferredArgument, true,
			[]string{"chain", "escape", "aggregate", "publication"},
		},
		{
			proofs.EvidenceProven, proofs.EvidenceDisproven, resourcepolicy.UnsummarizedCallee, false,
			[]string{"chain", "escape", "aggregate", "publication", "unmodeled"},
		},
	} {
		q := &queries{replies: map[string]resourcepolicy.Evidence{
			"aggregate":   {State: test.state, Reason: resourcepolicy.BudgetExhausted},
			"publication": {State: test.publication, Reason: resourcepolicy.BudgetExhausted},
		}}
		reason, opaque := resourcepolicy.Function(q, &ssa.Call{}, &ssa.CallCommon{Value: &ssa.Function{}}, true)
		if reason != test.reason || opaque != test.opaque || !reflect.DeepEqual(q.calls, test.calls) {
			t.Fatalf("state=%v reason=%v opaque=%v calls=%v", test.state, reason, opaque, q.calls)
		}
	}
}

func TestConsumptionKeepsFalsePayloadsAndStorageCutoffs(t *testing.T) {
	q := &queries{}
	if reason, opaque := resourcepolicy.Consume(q, &ssa.Return{}); reason != resourcepolicy.Untouched || opaque {
		t.Fatalf("return=%v/%v", reason, opaque)
	}
	q = &queries{replies: map[string]resourcepolicy.Evidence{"storage": {State: proofs.EvidenceUnknown, Reason: resourcepolicy.BudgetExhausted}}}
	if reason, opaque := resourcepolicy.Consume(q, &ssa.Store{}); reason != resourcepolicy.BudgetExhausted || !opaque ||
		!reflect.DeepEqual(q.calls, []string{"storage"}) {
		t.Fatalf("storage=%v/%v calls=%v", reason, opaque, q.calls)
	}
	q = &queries{}
	if reason, opaque := resourcepolicy.Send(q, nil, &ssa.Send{}); reason != resourcepolicy.SentToChannel || opaque ||
		!reflect.DeepEqual(q.calls, []string{"handoff", "payload"}) {
		t.Fatalf("send=%v/%v calls=%v", reason, opaque, q.calls)
	}
	if reason, opaque := resourcepolicy.Function(q, nil, &ssa.CallCommon{Value: &ssa.Parameter{}}, false); reason != resourcepolicy.DynamicCallee || opaque {
		t.Fatalf("dynamic=%v/%v", reason, opaque)
	}
}

func TestBuiltinFalseAnswersKeepTheInspectedReason(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "builtins", `package builtins
 func length(v []int)int{return len(v)}
 func appendOne(v []int)[]int{return append(v,1)}
 `)
	for _, name := range []string{"length", "appendOne"} {
		function := pkg.Func(name)
		for _, block := range function.Blocks {
			for _, i := range block.Instrs {
				call, ok := i.(*ssa.Call)
				if !ok {
					continue
				}
				if _, ok := call.Common().Value.(*ssa.Builtin); !ok {
					continue
				}
				for _, carried := range []bool{false, true} {
					reason, opaque := resourcepolicy.Dispatch(nil, call, call.Common(), carried)
					if reason != resourcepolicy.Appended || opaque != (name == "appendOne" && carried) {
						t.Fatalf("%s carried=%v: %v/%v", name, carried, reason, opaque)
					}
				}
			}
		}
	}
}
