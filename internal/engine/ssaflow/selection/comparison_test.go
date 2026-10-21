package selection_test

import (
	"go/constant"
	"go/token"
	"go/types"
	"testing"

	"github.com/kojah/gohawk/internal/engine/ssaflow/selection"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/ssa"
)

func TestSelectComparisonRecognition(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "example.com/dispatch", `package dispatch
 func Choose(a,b chan int){select{case <-a:case b<-1:default:}}
 `)
	var original *ssa.BinOp
	for _, block := range pkg.Func("Choose").Blocks {
		for _, instruction := range block.Instrs {
			if comparison, ok := instruction.(*ssa.BinOp); ok {
				original = comparison
				break
			}
		}
		if original != nil {
			break
		}
	}
	if original == nil {
		t.Fatal("no select dispatch comparison in SSA")
	}
	extracted, ok := original.X.(*ssa.Extract)
	if !ok {
		t.Fatal("dispatch does not extract an index")
	}
	selected, ok := extracted.Tuple.(*ssa.Select)
	if !ok {
		t.Fatal("dispatch does not refer to a select")
	}
	literal := func(n int64) ssa.Value { return ssa.NewConst(constant.MakeInt64(n), types.Typ[types.Int]) }
	for _, test := range []struct {
		name              string
		index             int
		op                token.Token
		number            int64
		reversed, matched bool
	}{
		{"equal", 0, token.EQL, 0, false, true},
		{"unequal", 0, token.NEQ, 1, false, true},
		{"reversed", 0, token.EQL, 1, true, true},
		{"reversed unequal", 0, token.NEQ, 0, true, true},
		{"default", 0, token.EQL, -1, false, true},
		{"bounds remain caller policy", 0, token.EQL, 99, false, true},
		{"receive ok is not index", 1, token.EQL, 0, false, false},
		{"payload is not index", 2, token.EQL, 0, false, false},
		{"ordering comparison", 0, token.LSS, 0, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			projection := *extracted
			projection.Index = test.index
			comparison := *original
			comparison.Op = test.op
			comparison.X = &projection
			comparison.Y = literal(test.number)
			if test.reversed {
				comparison.X, comparison.Y = comparison.Y, comparison.X
			}
			got, matched := selection.Recognize(&comparison)
			if matched != test.matched {
				t.Fatalf("matched=%v, want %v", matched, test.matched)
			}
			if matched && (got.Select != selected || got.Number != test.number || got.Equal != (test.op == token.EQL) || got.Reversed != test.reversed) {
				t.Fatalf("wrong structural evidence: %+v", got)
			}
		})
	}
	for _, value := range []ssa.Value{nil, original.X, literal(0)} {
		if _, matched := selection.Recognize(value); matched {
			t.Fatalf("recognized non-comparison %T", value)
		}
	}
	comparison := *original
	comparison.Y = comparison.X
	if _, matched := selection.Recognize(&comparison); matched {
		t.Fatal("nonconstant case recognized")
	}
	comparison.Y = ssa.NewConst(constant.MakeFloat64(1.5), types.Typ[types.Float64])
	if _, matched := selection.Recognize(&comparison); matched {
		t.Fatal("noninteger case recognized")
	}
}
