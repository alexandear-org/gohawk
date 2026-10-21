package resourcepolicy_test

import (
	"reflect"
	"testing"

	"github.com/kojah/gohawk/internal/analysis/analyzers/resources/resourcelifetime/resourcepolicy"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"golang.org/x/tools/go/ssa"
)

func (q *queries) Deferred(*ssa.Defer) resourcepolicy.Evidence { return q.evidence("deferred") }
func (q *queries) Guarded(ssa.Instruction, *ssa.MakeClosure) resourcepolicy.Evidence {
	return q.evidence("guarded")
}
func (q *queries) Owner(*ssa.MakeClosure) resourcepolicy.Evidence   { return q.evidence("owner") }
func (q *queries) Capture(*ssa.MakeClosure) resourcepolicy.Evidence { return q.evidence("capture") }
func (q *queries) Unreadable(*ssa.MakeClosure) bool {
	return q.evidence("unreadable").State == proofs.EvidenceProven
}

func (q *queries) Retains(*ssa.MakeClosure) bool {
	return q.evidence("retains").State == proofs.EvidenceProven
}

func TestClosureProofOrderAndTerminalUnknowns(t *testing.T) {
	cutoff := resourcepolicy.Evidence{State: proofs.EvidenceUnknown, Reason: resourcepolicy.BudgetExhausted}
	positive := resourcepolicy.Evidence{State: proofs.EvidenceProven, Reason: resourcepolicy.PriorDeferMayCleanCapturedCell}
	for _, test := range []struct {
		name        string
		instruction ssa.Instruction
		carried     bool
		replies     map[string]resourcepolicy.Evidence
		reason      resourcepolicy.Reason
		opaque      bool
		calls       []string
	}{
		{
			"deferred cutoff", &ssa.Defer{}, true,
			map[string]resourcepolicy.Evidence{"deferred": cutoff},
			resourcepolicy.BudgetExhausted, true,
			[]string{"deferred"},
		},
		{
			"deferred cleanup is uncertainty", &ssa.Defer{}, true,
			map[string]resourcepolicy.Evidence{"deferred": positive},
			resourcepolicy.PriorDeferMayCleanCapturedCell, true,
			[]string{"deferred"},
		},
		{
			"guarded cutoff", &ssa.Call{}, true,
			map[string]resourcepolicy.Evidence{"guarded": cutoff},
			resourcepolicy.BudgetExhausted, true,
			[]string{"guarded"},
		},
		{
			"positive guarded proof continues", &ssa.Call{}, true,
			map[string]resourcepolicy.Evidence{"guarded": positive},
			resourcepolicy.CapturedByRetainingLiteral, false,
			[]string{"guarded", "owner", "unreadable", "retains"},
		},
		{
			"owner cutoff precedes capture", &ssa.Call{}, false,
			map[string]resourcepolicy.Evidence{"owner": cutoff},
			resourcepolicy.BudgetExhausted, true,
			[]string{"guarded", "owner"},
		},
		{
			"possible owner precedes capture", &ssa.Call{}, false,
			map[string]resourcepolicy.Evidence{"owner": positive},
			resourcepolicy.PriorDeferMayCleanCapturedCell, true,
			[]string{"guarded", "owner"},
		},
		{
			"capture cutoff", &ssa.Call{}, false,
			map[string]resourcepolicy.Evidence{"capture": cutoff},
			resourcepolicy.BudgetExhausted, true,
			[]string{"guarded", "owner", "capture"},
		},
		{
			"uncarried launch is transparent", &ssa.Go{}, false, nil,
			resourcepolicy.None, false,
			[]string{"guarded", "owner", "capture"},
		},
		{
			"carried launch skips capture", &ssa.Go{}, true, nil,
			resourcepolicy.CapturedByStartedLiteral, true,
			[]string{"guarded", "owner"},
		},
		{
			"captured launch skips body queries", &ssa.Go{}, false,
			map[string]resourcepolicy.Evidence{"capture": positive},
			resourcepolicy.CapturedByStartedLiteral, true,
			[]string{"guarded", "owner", "capture"},
		},
		{
			"unreadable call stops retention", &ssa.Call{}, true,
			map[string]resourcepolicy.Evidence{"unreadable": positive},
			resourcepolicy.CapturedByLiteralCallingUnreadableCallee, true,
			[]string{"guarded", "owner", "unreadable"},
		},
		{
			"retained capture", &ssa.Call{}, false,
			map[string]resourcepolicy.Evidence{"capture": positive, "retains": positive},
			resourcepolicy.CapturedByRetainingLiteral, true,
			[]string{"guarded", "owner", "capture", "unreadable", "retains"},
		},
		{
			"deferred false payload", &ssa.Defer{}, true, nil,
			resourcepolicy.CapturedByRetainingLiteral, false,
			[]string{"deferred", "guarded", "owner", "unreadable", "retains"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			q := &queries{replies: test.replies}
			reason, opaque := resourcepolicy.Closure(q, test.instruction, &ssa.MakeClosure{}, test.carried)
			if reason != test.reason || opaque != test.opaque || !reflect.DeepEqual(q.calls, test.calls) {
				t.Fatalf("reason=%v opaque=%v calls=%v", reason, opaque, q.calls)
			}
		})
	}
}
