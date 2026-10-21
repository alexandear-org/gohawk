package lifecycleforms

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// FreshUseKind selects how the freshness walk handles one use of an acquired
// resource. Follow and cleanup remain requests for host evidence, not claims.
type FreshUseKind uint8

const (
	FreshReject FreshUseKind = iota
	FreshIgnore
	FreshFollow
	FreshReturn
	FreshCleanup
)

// FreshUse retains the exact node needed by the enclosing return/cleanup proof.
type FreshUse struct {
	Kind   FreshUseKind
	Alias  ssa.Value
	Return *ssa.Return
	Call   *ssa.Call
}
type freshSignature func(ssa.Instruction) FreshUse

var _ freshSignature = Freshness

const (
	equalToken    = token.EQL
	notEqualToken = token.NEQ
)
