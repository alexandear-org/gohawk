package lifecycleforms

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// HandoffUseKind selects how a direct or wrapped result walk handles a use.
// Following an alias or recognizing a call requests host evidence, not ownership.
type HandoffUseKind uint8

const (
	HandoffReject HandoffUseKind = iota
	HandoffIgnore
	HandoffFollow
	HandoffReturn
	HandoffCall
)

// HandoffUse retains the exact node needed by the enclosing handoff proof.
type HandoffUse struct {
	Kind   HandoffUseKind
	Alias  ssa.Value
	Return *ssa.Return
	Call   *ssa.Call
}
type handoffSignature func(ssa.Instruction) HandoffUse

var _ handoffSignature = Handoff

const (
	equalToken    = token.EQL
	notEqualToken = token.NEQ
)
