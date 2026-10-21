package resourcelifetime

import (
	"testing"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"golang.org/x/tools/go/ssa"
)

func TestResourceClassifierEvidenceOrder(t *testing.T) {
	for _, test := range []struct {
		name, body string
		action     resourceAction
		reason     resourceLifetimeReason
	}{
		{"release", "value.Close()", actionSettled, resourceReasonSettled},
		{"borrow", "println(value)", actionNone, resourceReasonUntouched},
		{"conditional", "if step(0)>0 {value.Close()}", actionNone, resourceReasonUntouched},
		{"loop", "for step(0)>0 {value.Close()}", actionUnknown, resourceReasonHelperCleanupInLoop},
	} {
		t.Run(test.name, func(t *testing.T) {
			query, call := helperBudgetAnalysis(t, test.body)
			for _, limit := range []int{0, 1, 5, resourcePoolBudget} {
				query.pool = proofs.NewSearchBudget(limit)
				action, reason := query.classify(call)
				wantAction, wantReason := test.action, test.reason
				cutoff := limit != resourcePoolBudget
				if cutoff {
					wantAction, wantReason = actionUnknown, resourceReasonBudgetExhausted
				}
				if action != wantAction || reason != wantReason || query.pool.Exhausted() != cutoff {
					t.Fatalf("limit=%d got %v/%v exhausted=%v, want %v/%v cutoff=%v",
						limit, action, reason, query.pool.Exhausted(), wantAction, wantReason, cutoff)
				}
			}
			query.pool = proofs.NewSearchBudget(0)
			query.collection = &localCollection{appends: []*ssa.Call{call}}
			if action, reason := query.classify(call); action != actionNone || reason != resourceReasonAppendedToLocalCollection || query.pool.Exhausted() {
				t.Fatalf("collection label lost precedence: %v/%v exhausted=%v", action, reason, query.pool.Exhausted())
			}
		})
	}
}
