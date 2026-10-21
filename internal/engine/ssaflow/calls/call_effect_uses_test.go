package calls

import (
	"testing"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	ssaflow "github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/ssa"
)

// Exercise the public proof rather than a second implementation of the use
// classifier: alias recursion, call dispatch and completeness all matter here.
func TestCallEffectUseBoundaries(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "effectuses", `package effectuses
 type box struct{x, y int}
 type integer int
 type first interface{First(); Second()}
 type second interface{First()}
 var saved *int
 func read(p *int){_ = *p}
 func write(p *int){*p=1}
 func comparison(p *int)bool{return p==nil}
 func returned(p *int)*int{return p}
 func stored(p *int){saved=p}
 func sent(p *int, ch chan *int){ch<-p}
 func channel(ch chan int){ch<-1}
 func mapKey(p *int, m map[*int]int){m[p]=1}
 func mapValue(p *int, m map[int]*int){m[0]=p}
 func mapObject(m map[int]int){m[0]=1}
 func field(p *box){p.x=1}
 func index(p []int){p[0]=1}
 func slice(p []int){q:=p[:]; q[0]=1}
 func changedType(p *int){q:=(*integer)(p); *q=1}
 func boxed(p *int)any{return p}
 func changedInterface(p first)second{return p}
 func merged(p *box, choose bool){q:=new(box); if choose{q=p}; q.x=1}
 func snapshot(p **int){q:=*p; *q=1}
 func receive(ch chan int){_ = <-ch}
 func arithmetic(p int)int{return p+1}
 func fieldValue(p box)int{return p.x}
 func throughCall(p *int){read(p)}
 func deferred(p *int){defer read(p)}
 func launched(p *int){go read(p)}
 func closure(p *int){func(){*p=1}()}
 func invoked(p func()){p()}
 `)
	for _, test := range []struct {
		name     string
		effects  CallEffect
		complete bool
	}{
		{"read", EffectRead, true},
		{"write", EffectMutate, true},
		{"comparison", EffectRead, true},
		{"returned", EffectRetain, true},
		{"stored", EffectRetain, false},
		{"sent", EffectRetain | EffectAsync, false},
		{"channel", EffectMutate, true},
		{"mapKey", EffectRetain, false},
		{"mapValue", EffectRetain, false},
		{"mapObject", EffectMutate, true},
		{"field", EffectMutate, true},
		{"index", EffectMutate, true},
		{"slice", EffectMutate, true},
		{"changedType", EffectMutate, true},
		{"boxed", EffectRetain, true},
		{"changedInterface", EffectRetain, true},
		{"merged", EffectMutate, true},
		{"snapshot", EffectRead, true},
		{"receive", 0, false},
		{"arithmetic", 0, false},
		{"fieldValue", EffectRetain, false},
		{"throughCall", EffectRead, true},
		{"deferred", EffectRead, true},
		{"launched", EffectRead | EffectAsync, true},
		{"closure", EffectRetain, false},
		{"invoked", EffectInvoke, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			proof := NewCallEffects(proofs.NewSearchBudget(1000)).Value(pkg.Func(test.name).Params[0])
			if proof.Effects != test.effects || proof.Proven() != test.complete {
				t.Fatalf("got %+v, want effects=%v complete=%v", proof, test.effects, test.complete)
			}
		})
	}
	// SSA spills these parameters into local cells before capture or field
	// access. That publication remains unknown rather than inferring purity.
	closure := ssaflow.InstructionsOf[*ssa.MakeClosure](pkg.Func("closure"))[0]
	captured := NewCallEffects(proofs.NewSearchBudget(1000))
	proof := captured.proof(captured.use(closure.Bindings[0], closure, make(map[ssa.Value]bool)))
	if !proof.Proven() || proof.Effects != EffectRead {
		t.Fatalf("loaded capture became an alias of its pointee: %+v", proof)
	}
	query := NewCallEffects(proofs.NewSearchBudget(1))
	value := pkg.Func("field").Params[0]
	cut := query.Value(value)
	if cut.Proven() || cut.Reason != proofs.EvidenceBudgetExhausted {
		t.Fatalf("exhausted alias walk established completeness: %+v", cut)
	}
	query.budget = proofs.NewSearchBudget(1000)
	fresh := query.Value(value)
	if !fresh.Proven() || fresh.Effects != EffectMutate {
		t.Fatalf("fresh allowance did not recover alias mutation: %+v", fresh)
	}
}

func TestCallEffectFieldUseBoundaries(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "fieldeffects", `package fieldeffects
 type box struct{x, y int; pointer *int}
 type outer struct{padding int; child box}
 var saved *box
 func read(p *box){_ = p.x}
 func other(p *box){p.y=1}
 func write(p *box){p.x=1}
 func replace(p *box){*p=box{}}
 func publish(p *box){saved=p}
 func snapshot(p *box){q:=p.pointer; *q=1}
 func compare(p *box)bool{return p==nil}
 func nested(p *outer){p.child.y=1}
 func nestedCall(p *outer){nested(p)}
 func readCall(p *box){read(p)}
 func otherCall(p *box){other(p)}
 func writeCall(p *box){write(p)}
 func replaceCall(p *box){replace(p)}
 func publishCall(p *box){publish(p)}
 func snapshotCall(p *box){snapshot(p)}
 func compareCall(p *box){compare(p)}
 `)
	for _, test := range []struct {
		name     string
		fields   []int
		effects  CallEffect
		complete bool
	}{
		{"read", []int{0}, EffectRead, true},
		{"other", []int{0}, 0, true},
		{"write", []int{0}, EffectMutate, true},
		{"replace", []int{0}, EffectMutate, true},
		{"publish", []int{0}, EffectRetain, true},
		{"snapshot", []int{2}, EffectRead, true},
		{"compare", []int{0}, 0, false},
		{"other", nil, EffectMutate, true},
		{"nested", []int{1, 1}, EffectMutate, true},
		{"nested", []int{1, 0}, 0, true},
		{"nested", []int{0}, 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := pkg.Func(test.name + "Call")
			call := ssaflow.InstructionsOf[*ssa.Call](function)[0]
			path := ssaflow.EmbeddedFieldPath{Root: function.Params[0], Depth: len(test.fields)}
			copy(path.Fields[:], test.fields)
			proof := NewCallEffects(proofs.NewSearchBudget(1000)).FieldCall(call, path)
			if proof.Effects != test.effects || proof.Proven() != test.complete {
				t.Fatalf("got %+v, want effects=%v complete=%v", proof, test.effects, test.complete)
			}
		})
	}
}
