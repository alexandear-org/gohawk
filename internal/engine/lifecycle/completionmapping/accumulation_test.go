package completionmapping_test

import (
	"testing"

	"github.com/kojah/gohawk/internal/engine/lifecycle/completionmapping"
)

func TestCompletionPathsRetainOnlyUnanimousEvidence(t *testing.T) {
	root := completionmapping.Paths{Seen: true, Exact: true}
	out := completionmapping.Paths{Path: "out", Seen: true, Exact: true}
	other := completionmapping.Paths{Path: "other", Seen: true, Exact: true}
	unknown := completionmapping.Paths{Seen: true}
	inexactOut := completionmapping.Paths{Path: "out", Seen: true}
	for _, test := range []struct {
		name  string
		calls []completionmapping.Paths
		want  completionmapping.Paths
	}{
		{"no observations", []completionmapping.Paths{{}}, completionmapping.Paths{}},
		{"root is a named path", []completionmapping.Paths{root}, root},
		{"same field", []completionmapping.Paths{out, out}, out},
		{"sibling invalidates", []completionmapping.Paths{out, other, out}, inexactOut},
		{"unknown invalidates", []completionmapping.Paths{out, unknown, out}, inexactOut},
		{"unknown before named", []completionmapping.Paths{unknown, out}, unknown},
		{"unseen payload is ignored", []completionmapping.Paths{out, {Path: "other", Exact: true}}, out},
	} {
		t.Run(test.name, func(t *testing.T) {
			var got completionmapping.Paths
			for _, call := range test.calls {
				got.Merge(call)
			}
			if got != test.want || got.Known() != (test.want.Seen && test.want.Exact) {
				t.Fatalf("paths=%+v known=%v, want %+v", got, got.Known(), test.want)
			}
		})
	}
	var ignored *completionmapping.Paths
	ignored.Record("out", true)
	ignored.Merge(completionmapping.Paths{Path: "out", Seen: true, Exact: true})
}
