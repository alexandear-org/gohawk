package recognition_test

import (
	"go/token"
	"testing"

	"github.com/kojah/gohawk/internal/analysis/passes/concurrencyfacts/recognition"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/ssa"
)

func TestContextRecognitionKeepsExactInputs(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "contextcases", `package contextcases
import "context"
type CancelFunc func()
type mimic struct{}
func (*mimic) Done()<-chan struct{}{return nil}
func Background()context.Context{return context.Background()}
func WithCancel(p context.Context)(context.Context,context.CancelFunc){return context.WithCancel(p)}
func fakeFresh(p context.Context)(context.Context,context.CancelFunc){return WithCancel(p)}
func empty(){_ = context.Background()}
func todo(){_ = context.TODO()}
func fresh(p context.Context)(context.Context,context.CancelFunc){return context.WithCancel(p)}
func cause(p context.Context)(context.Context,context.CancelCauseFunc){return context.WithCancelCause(p)}
func observe(p context.Context)<-chan struct{}{return p.Done()}
func request(p context.CancelFunc){p()}
func requestCause(p context.CancelCauseFunc){p(nil)}
func fakeDone(p *mimic){_ = p.Done()}
func fakeBackground(){_ = Background()}
func opaque(p CancelFunc){p()}
`)
	for _, test := range []struct {
		name  string
		kind  recognition.ContextKind
		input bool
	}{
		{"empty", recognition.ContextEmpty, false},
		{"todo", recognition.ContextEmpty, false},
		{"fresh", recognition.ContextFresh, true},
		{"cause", recognition.ContextFresh, true},
		{"observe", recognition.ContextObserve, true},
		{"request", recognition.ContextRequest, true},
		{"requestCause", recognition.ContextRequest, true},
		{"fakeDone", recognition.ContextUnknown, false},
		{"fakeBackground", recognition.ContextUnknown, false},
		{"fakeFresh", recognition.ContextUnknown, false},
		{"opaque", recognition.ContextUnknown, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := pkg.Func(test.name)
			call := ssaflow.InstructionsOf[*ssa.Call](function)[0]
			kind, value := recognition.ContextCall(call.Common())
			var want ssa.Value
			if test.input {
				want = function.Params[0]
			}
			if kind != test.kind || value != want {
				t.Fatalf("%s: got %v/%v, want %v/%v", call, kind, value, test.kind, want)
			}
		})
	}
	for _, name := range []string{"fresh", "cause"} {
		function := pkg.Func(name)
		call := ssaflow.InstructionsOf[*ssa.Call](function)[0]
		for _, result := range ssaflow.InstructionsOf[*ssa.Extract](function) {
			kind, origin, observed := recognition.ContextOrigin(result)
			if kind != recognition.OriginFresh || origin != call || observed != nil {
				t.Fatalf("constructor projection %s: %v/%v/%v", result, kind, origin, observed)
			}
		}
		if kind, _, _ := recognition.ContextOrigin(call); kind != recognition.OriginUnknown {
			t.Fatalf("raw constructor tuple became an origin: %v", kind)
		}
	}
	call := ssaflow.InstructionsOf[*ssa.Call](pkg.Func("observe"))[0]
	if kind, origin, value := recognition.ContextOrigin(call); kind != recognition.OriginObserved || origin != nil || value != pkg.Func("observe").Params[0] {
		t.Fatalf("Done projection: %v/%v/%v", kind, origin, value)
	}
	if kind, value := recognition.ContextCall(nil); kind != recognition.ContextUnknown || value != nil {
		t.Fatalf("missing context call: %v/%v", kind, value)
	}
}

func TestCallbackRecognitionScope(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "callbackcases", `package callbackcases
import "context"
type runner interface{Run(int)}
func ordinary(f func(int)){f(1)}
func method(p runner){p.Run(1)}
func contextMethod(p context.Context){_ = p.Done()}
func channelArg(f func(chan int), ch chan int){f(ch)}
func indirect(p *func()){(*p)()}
func target(){}
func direct(){target()}
func launched(f func()){go f()}
func deferred(f func()){defer f()}
`)
	for _, test := range []struct {
		name       string
		recognized bool
	}{
		{"ordinary", true},
		{"method", true},
		{"contextMethod", false},
		{"channelArg", false},
		{"indirect", false},
		{"direct", false},
		{"launched", false},
		{"deferred", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := pkg.Func(test.name)
			var instruction ssa.CallInstruction
			switch test.name {
			case "launched":
				instruction = ssaflow.InstructionsOf[*ssa.Go](function)[0]
			case "deferred":
				instruction = ssaflow.InstructionsOf[*ssa.Defer](function)[0]
			default:
				instruction = ssaflow.InstructionsOf[*ssa.Call](function)[0]
			}
			value, method, recognized := recognition.CallbackCall(instruction)
			if recognized != test.recognized {
				t.Fatalf("%s: recognized=%v, want %v", instruction, recognized, test.recognized)
			}
			if recognized {
				if value != function.Params[0] || method != instruction.Common().Method || !recognition.CallbackInput(value, method) {
					t.Fatalf("callback lost exact input/method: %v/%v", value, method)
				}
			} else if value != nil || method != nil {
				t.Fatalf("unrecognized callback returned evidence: %v/%v", value, method)
			}
		})
	}
	if _, _, recognized := recognition.CallbackCall(nil); recognized || recognition.CallbackInput(nil, nil) {
		t.Fatal("missing callback input was admitted")
	}
}

func TestResourceReferenceRetainsRejectedCallAndExactInputs(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "referencecases", `package referencecases
 import("context";"sync")
 var shared sync.Mutex
 type holder struct{mu sync.Mutex}
 func NewCond(mu *sync.Mutex)*sync.Cond{return sync.NewCond(mu)}
 func exact(mu *sync.Mutex)*sync.Cond{return sync.NewCond(mu)}
 func mimic(mu *sync.Mutex)*sync.Cond{return NewCond(mu)}
 func inputs(mu *sync.Mutex,ctx context.Context,cancel context.CancelFunc,ch chan int){}
 func field(p *holder)*sync.Mutex{return &p.mu}
 func local()*sync.Mutex{return new(sync.Mutex)}
 func ordinary()*int{return new(int)}
 `)
	for _, tt := range []struct {
		name string
		kind recognition.ReferenceKind
	}{
		{"exact", recognition.ReferenceExact}, {"mimic", recognition.ReferenceRejected},
	} {
		call := ssaflow.InstructionsOf[*ssa.Call](pkg.Func(tt.name))[0]
		kind, value, cancellation := recognition.ResourceReference(call)
		if kind != tt.kind || value != call || cancellation {
			t.Fatalf("%s: kind=%v value=%v cancellation=%v", tt.name, kind, value, cancellation)
		}
	}
	for i, parameter := range pkg.Func("inputs").Params {
		kind, value, cancellation := recognition.ResourceReference(parameter)
		if kind != recognition.ReferenceExact || value != parameter || cancellation != (i == 1 || i == 2) {
			t.Fatalf("%s: kind=%v value=%v cancellation=%v", parameter, kind, value, cancellation)
		}
	}
	field := ssaflow.InstructionsOf[*ssa.FieldAddr](pkg.Func("field"))[0]
	global := pkg.Members["shared"].(*ssa.Global)
	allocation := ssaflow.InstructionsOf[*ssa.Alloc](pkg.Func("local"))[0]
	ordinary := ssaflow.InstructionsOf[*ssa.Alloc](pkg.Func("ordinary"))[0]
	for _, tt := range []struct {
		input ssa.Value
		kind  recognition.ReferenceKind
	}{
		{field, recognition.ReferenceAddress},
		{global, recognition.ReferenceAddress},
		{allocation, recognition.ReferenceExact},
		{ordinary, recognition.ReferenceUnknown},
	} {
		kind, value, cancellation := recognition.ResourceReference(tt.input)
		want := tt.input
		if tt.kind == recognition.ReferenceUnknown {
			want = nil
		}
		if kind != tt.kind || value != want || cancellation {
			t.Fatalf("%s: kind=%v value=%v cancellation=%v", tt.input, kind, value, cancellation)
		}
	}
}

const rootSelectorFixture = `package rootcases
 import "sync"
 var mu sync.Mutex
 type holder struct{mu *sync.Mutex}
 func field(p *holder)*sync.Mutex{return p.mu}
 func capture(ch chan *sync.Mutex)func()*sync.Mutex{return func()*sync.Mutex{return <-ch}}
 `

func TestRootSelectorsKeepFieldAndGlobalAddresses(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "rootcases", rootSelectorFixture)
	fieldLoad := ssaflow.InstructionsOf[*ssa.UnOp](pkg.Func("field"))[0]
	field := fieldLoad.X.(*ssa.FieldAddr)
	bindingRoot := func(load *ssa.UnOp) (recognition.AddressKind, *ssa.FreeVar, *ssa.FieldAddr) {
		return recognition.BindingRoot(load)
	}
	for _, choose := range []recognition.FixedLoadSignature{recognition.FixedLoad, bindingRoot} {
		kind, capture, address := choose(fieldLoad)
		if kind != recognition.AddressField || capture != nil || address != field {
			t.Fatalf("field root=%v/%v/%v", kind, capture, address)
		}
	}
	global := pkg.Members["mu"].(*ssa.Global)
	if admitted, load := recognition.PathRoot(global, true); !admitted || load != nil {
		t.Fatal("global mutex address lost")
	}
	if admitted, load := recognition.PathRoot(global, false); admitted || load != nil {
		t.Fatal("global content became an exact root")
	}
}

func TestRootSelectorsKeepCaptureAndReceiveEvidenceSeparate(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "rootcases", rootSelectorFixture)
	bindingRoot := func(load *ssa.UnOp) (recognition.AddressKind, *ssa.FreeVar, *ssa.FieldAddr) {
		return recognition.BindingRoot(load)
	}
	// SSA reads the captured channel cell before receiving. Only that cell
	// read is a fixed load; the receive needs separate path evidence.
	var captured, received *ssa.UnOp
	var capture *ssa.FreeVar
	for _, unary := range ssaflow.InstructionsOf[*ssa.UnOp](pkg.Func("capture").AnonFuncs[0]) {
		if unary.Op == token.ARROW {
			received = unary
		}
		if source, ok := unary.X.(*ssa.FreeVar); ok && unary.Op == token.MUL {
			captured, capture = unary, source
		}
	}
	if captured == nil || received == nil {
		t.Fatal("missing captured channel read or receive")
	}
	for _, choose := range []recognition.FixedLoadSignature{recognition.FixedLoad, bindingRoot} {
		if kind, got, address := choose(captured); kind != recognition.AddressCapture || got != capture || address != nil {
			t.Fatalf("captured cell=%v/%v/%v", kind, got, address)
		}
	}
	if kind, got, address := recognition.FixedLoad(received); kind != recognition.AddressBlocked || got != nil || address != nil {
		t.Fatalf("receive became fixed load=%v/%v/%v", kind, got, address)
	}
	if admitted, load := recognition.PathRoot(received, false); admitted || load != received {
		t.Fatal("unary root lost its deferred proof")
	}
}
