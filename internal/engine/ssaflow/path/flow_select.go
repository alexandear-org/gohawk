package path

import (
	"go/types"

	"github.com/kojah/gohawk/internal/engine/ssaflow/selection"
	"golang.org/x/tools/go/ssa"
)

// SelectedReceiveChannel returns the channel necessarily received from before
// entering block. The predecessor must select that exact receive case through
// SSA's select index; a default, send case, or shared successor proves nothing.
// This only describes the selected operation, not its synchronization policy.
func SelectedReceiveChannel(block *ssa.BasicBlock) (ssa.Value, bool) { //nolint:ireturn // Channels retain concrete SSA identities.
	if block == nil || len(block.Preds) != 1 {
		return nil, false
	}
	return SelectedReceiveOnEdge(block.Preds[0], block)
}

// SelectedReceiveOnEdge returns the channel received from when the exact
// select-case edge is taken. Other predecessors of to may establish no receive;
// callers must keep this evidence on the edge, not on the shared destination.
func SelectedReceiveOnEdge(from, to *ssa.BasicBlock) (ssa.Value, bool) { //nolint:ireturn // Channels retain concrete SSA identities.
	if from == nil || to == nil || len(from.Instrs) == 0 || len(from.Succs) != 2 || from.Succs[0] != to {
		return nil, false
	}
	branch, ok := from.Instrs[len(from.Instrs)-1].(*ssa.If)
	if !ok {
		return nil, false
	}
	// Preserve this query's canonical-equality boundary. The shared recognizer
	// also serves summary dispatch, whose policy accepts reversed and != forms.
	match, ok := selection.Recognize(branch.Cond)
	if !ok || !match.Equal || match.Reversed || match.Number < 0 || match.Number >= int64(len(match.Select.States)) {
		return nil, false
	}
	state := match.Select.States[match.Number]
	return state.Chan, state.Dir == types.RecvOnly
}
