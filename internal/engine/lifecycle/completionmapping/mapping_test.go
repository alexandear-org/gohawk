package completionmapping_test

import (
	"reflect"
	"testing"

	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/lifecycle/completionmapping"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/ssa"
)

type queryRecorder struct {
	calls    []string
	callback bool
}

func (queries *queryRecorder) Callback(ssa.Value, ssa.Value) bool {
	queries.calls = append(queries.calls, "callback")
	return queries.callback
}

func (queries *queryRecorder) MayContain(ssa.Value, ssa.Value) bool {
	queries.calls = append(queries.calls, "containment")
	return true
}

func (queries *queryRecorder) DirectStore(ssa.Value) bool {
	queries.calls = append(queries.calls, "store")
	return false
}

func (queries *queryRecorder) DeferredValue(ssa.Value, ssa.Value, ssa.Instruction) heapmodel.StoredValue {
	queries.calls = append(queries.calls, "deferred")
	return heapmodel.StoredValue{}
}

func (queries *queryRecorder) StoredOnPath(ssa.Value, ssa.Value, ssa.Instruction) bool {
	queries.calls = append(queries.calls, "stored-path")
	return false
}

func (queries *queryRecorder) Uncertain() (completionmapping.Local, bool) {
	queries.calls = append(queries.calls, "uncertain")
	return completionmapping.Local{}, false
}

func (queries *queryRecorder) Cutoff() (completionmapping.Local, bool) {
	queries.calls = append(queries.calls, "cutoff")
	return completionmapping.Local{}, false
}

func TestExactCaptureKeepsUnavailablePayload(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "exactcapture", `package exactcapture
 func inputs(first,second *int){}
 `)
	first, second := pkg.Func("inputs").Params[0], pkg.Func("inputs").Params[1]
	queries := &queryRecorder{}
	context := completionmapping.Context{Queries: queries, Exact: true}
	local, ok := completionmapping.Captured(context, first, first, second, nil)
	if ok || local.Local != first || local.Supplied != first || local.Kind != completionmapping.Exact || local.Path != nil || len(queries.calls) != 0 {
		t.Fatalf("mapping=%+v available=%v queries=%v", local, ok, queries.calls)
	}
}

func TestCaptureRelationCommitsItsSelectedEvidence(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "capturerelation", `package capturerelation
 type holder struct{field int}
 func field(owner *holder,unrelated *int)*int{return &owner.field}
 `)
	function := pkg.Func("field")
	binding, value := function.Params[0], function.Params[1]
	target := ssaflow.InstructionsOf[*ssa.FieldAddr](function)[0]
	for _, callback := range []bool{false, true} {
		t.Run(map[bool]string{false: "unavailable-owner", true: "callback"}[callback], func(t *testing.T) {
			queries := &queryRecorder{callback: callback}
			context := completionmapping.Context{Queries: queries, Budget: proofs.NewSearchBudget(proofs.QueryBudget)}
			local, ok := completionmapping.CaptureRelation(context, value, binding, target, value, false, false, nil)
			if ok != callback || !reflect.DeepEqual(queries.calls, []string{"callback"}) {
				t.Fatalf("mapping=%+v available=%v queries=%v", local, ok, queries.calls)
			}
			if !callback && !reflect.DeepEqual(local, completionmapping.Local{}) {
				t.Fatalf("unavailable captured owner leaked a mapping: %+v", local)
			}
			if callback && (local.Kind != completionmapping.Callback || local.Local != value || local.Supplied != value) {
				t.Fatalf("callback mapping changed identity: %+v", local)
			}
		})
	}
}

func TestPathTranslationPreservesAvailabilityAndNilOwnerPaths(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "translation", `package translation
 func input(value *int){}
 `)
	value := pkg.Func("input").Params[0]
	for _, test := range []struct {
		name                 string
		kind                 completionmapping.Kind
		expected, actual     []string
		known                bool
		allowance            int
		path                 string
		available, exhausted bool
	}{
		{"direct-owner", completionmapping.Owner, nil, nil, true, 0, "", true, false},
		{"empty-nested-owner", completionmapping.Owner, nil, []string{}, true, 0, "", false, true},
		{"known-empty-owner", completionmapping.Owner, nil, []string{}, true, proofs.QueryBudget, "", true, false},
		{"unknown-owner", completionmapping.Owner, nil, nil, false, 0, "", false, false},
		{"contained-exact", completionmapping.Exact, []string{"field:0"}, []string{"field:0"}, true, 0, "", true, false},
		{"sibling-exact", completionmapping.Exact, []string{"field:0"}, []string{"field:1"}, true, 0, "", false, false},
		{"exact-forward", completionmapping.Exact, nil, []string{"field:0"}, true, proofs.QueryBudget, "field:0", true, false},
		{"projection", completionmapping.Projection, []string{"field:0"}, nil, true, 0, "field:0", true, false},
		{"projection-child", completionmapping.Projection, []string{"field:0"}, []string{"field:0"}, true, 0, "", false, false},
		{"projection-cutoff", completionmapping.Projection, nil, nil, true, 0, "", false, true},
		{"callback", completionmapping.Callback, nil, nil, true, 0, "", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			local := completionmapping.Local{Kind: test.kind, Supplied: value, Path: test.expected}
			budget := proofs.NewSearchBudget(test.allowance)
			path, available := completionmapping.TranslatePath(local, value, test.actual, test.known, budget)
			if path != test.path || available != test.available || budget.Exhausted() != test.exhausted {
				t.Fatalf("got %q/%v exhausted=%v; want %q/%v exhausted=%v", path, available, budget.Exhausted(), test.path, test.available, test.exhausted)
			}
		})
	}
}
