package valueforms_test

import (
	"testing"

	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"github.com/kojah/gohawk/internal/engine/ssaflow/valueforms"
	"golang.org/x/tools/go/ssa"
)

func TestOwnershipSourceKeepsOneStepIdentity(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "sources", `package sources
type holder struct{C chan int}
type messages chan int
func field(p *holder)chan int{return p.C}
func identity(p holder)holder{return p}
func record(p holder)chan int{return identity(p).C}
func indexed(p []chan int)chan int{return p[0]}
func text(p string)byte{return p[0]}
func sliced(p []chan int)[]chan int{return p[1:]}
func lookup(p map[string]chan int)(chan int,bool){v,ok:=p["x"];return v,ok}
func asserted(p any)(chan int,bool){v,ok:=p.(chan int);return v,ok}
func iterated(p map[string]chan int)chan int{for _,v:=range p{return v};return nil}
func received(p chan messages)messages{return <-p}
func negated(p int)int{return -p}
func inverted(p bool)bool{return !p}
func allocated()*holder{return new(holder)}
`)
	parameter := func(function *ssa.Function) ssa.Value { return function.Params[0] }
	for _, test := range []struct {
		name, function string
		value          func(*ssa.Function) ssa.Value
		source         func(*ssa.Function) ssa.Value
		recognized     bool
		external       bool
	}{
		{"field address", "field", firstInstruction[*ssa.FieldAddr], parameter, true, true},
		{"load stops at cell", "field", firstInstruction[*ssa.UnOp], firstInstruction[*ssa.FieldAddr], true, true},
		{"field value", "record", firstInstruction[*ssa.Field], firstInstruction[*ssa.Call], true, false},
		{"index address", "indexed", firstInstruction[*ssa.IndexAddr], parameter, true, true},
		{"index value", "text", firstInstruction[*ssa.Index], parameter, true, true},
		{"slice", "sliced", firstInstruction[*ssa.Slice], parameter, true, true},
		{"lookup", "lookup", firstInstruction[*ssa.Lookup], parameter, true, true},
		{"lookup extraction", "lookup", firstInstruction[*ssa.Extract], firstInstruction[*ssa.Lookup], true, true},
		{"assertion", "asserted", firstInstruction[*ssa.TypeAssert], parameter, true, true},
		{"assertion extraction", "asserted", firstInstruction[*ssa.Extract], firstInstruction[*ssa.TypeAssert], true, true},
		{"range", "iterated", firstInstruction[*ssa.Range], parameter, true, true},
		{"next", "iterated", firstInstruction[*ssa.Next], firstInstruction[*ssa.Range], true, true},
		{"receive", "received", firstInstruction[*ssa.UnOp], parameter, true, true},
		{"negation has no ownership edge", "negated", firstInstruction[*ssa.UnOp], parameter, false, false},
		{"logical operation has no ownership edge", "inverted", firstInstruction[*ssa.UnOp], parameter, false, false},
		{"parameter stops", "field", parameter, nil, false, true},
		{"allocation stops", "allocated", firstInstruction[*ssa.Alloc], nil, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := pkg.Func(test.function)
			value := test.value(function)
			var want ssa.Value
			if test.source != nil {
				want = test.source(function)
			}
			source, recognized := valueforms.OwnershipSource(value)
			if source != want || recognized != test.recognized {
				t.Fatalf("%s: source=%v recognized=%v, want %v/%v", value, source, recognized, want, test.recognized)
			}
			// The reaching walk retains its own ownership policy. Arithmetic and
			// opaque call results cannot turn a parameter into a proven owner.
			if ssaflow.ExternallyOwnedValue(value) != test.external {
				t.Fatalf("%s: external ownership did not preserve the source boundary", value)
			}
		})
	}
}

func firstInstruction[T interface {
	ssa.Value
	ssa.Instruction
}](function *ssa.Function) ssa.Value {
	return ssaflow.InstructionsOf[T](function)[0]
}
