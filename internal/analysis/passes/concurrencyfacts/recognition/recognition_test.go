package recognition_test

import (
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
