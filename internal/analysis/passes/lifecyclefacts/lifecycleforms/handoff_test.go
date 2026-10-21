package lifecycleforms

import (
	"go/token"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestHandoffPreservesFormsAndIdentity(t *testing.T) {
	returned := &ssa.Return{}
	call := &ssa.Call{}
	boxed := &ssa.MakeInterface{}
	changed := &ssa.ChangeInterface{}
	merged := &ssa.Phi{}
	for _, scenario := range []struct {
		name string
		use  ssa.Instruction
		want HandoffUse
	}{
		{"return", returned, HandoffUse{Kind: HandoffReturn, Return: returned}},
		{"call", call, HandoffUse{Kind: HandoffCall, Call: call}},
		{"box", boxed, HandoffUse{Kind: HandoffFollow, Alias: boxed}},
		{"interface", changed, HandoffUse{Kind: HandoffFollow, Alias: changed}},
		{"phi", merged, HandoffUse{Kind: HandoffFollow, Alias: merged}},
		{"debug", &ssa.DebugRef{}, HandoffUse{Kind: HandoffIgnore}},
		{"equal", &ssa.BinOp{Op: token.EQL}, HandoffUse{Kind: HandoffIgnore}},
		{"not equal", &ssa.BinOp{Op: token.NEQ}, HandoffUse{Kind: HandoffIgnore}},
		{"ordered comparison", &ssa.BinOp{Op: token.LSS}, HandoffUse{}},
		{"arithmetic", &ssa.BinOp{Op: token.ADD}, HandoffUse{}},
		{"load", &ssa.UnOp{Op: token.MUL}, HandoffUse{}},
		{"conversion", &ssa.Convert{}, HandoffUse{}},
		{"retyping", &ssa.ChangeType{}, HandoffUse{}},
		{"store", &ssa.Store{}, HandoffUse{}},
		{"capture", &ssa.MakeClosure{}, HandoffUse{}},
		{"defer", &ssa.Defer{}, HandoffUse{}},
		{"launch", &ssa.Go{}, HandoffUse{}},
		{"nil", nil, HandoffUse{}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			if got := Handoff(scenario.use); got != scenario.want {
				t.Errorf("handoff = %+v, want %+v", got, scenario.want)
			}
		})
	}
}
