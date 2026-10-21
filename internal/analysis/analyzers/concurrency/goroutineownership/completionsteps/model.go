// Package completionsteps recognizes the structural instructions permitted after
// worker completion or opaque work. The analyzer owns CFG traversal, cycle
// rejection, allowance charges and validation of deferred completion operations.
package completionsteps

import (
	"go/token"

	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// TerminalStep tells the terminal-completion walk how to advance. Defers still
// require the original Done origin and a complete deferred-operation census.
type TerminalStep uint8

const (
	Reject TerminalStep = iota
	Returned
	RunDefers
	Follow
)

// TailStep identifies an instruction after opaque work. A deferred-operation
// plan does not itself establish that all deferred calls are completion-only.
type TailStep uint8

const (
	Work TailStep = iota
	Passive
	Deferred
)

type (
	terminalSignature func(ssa.Instruction) TerminalStep
	tailSignature     func(ssa.Instruction) TailStep
)

var (
	_ terminalSignature = Terminal
	_ tailSignature     = Tail
)

const loadToken = token.MUL

func closes(call *ssa.Call) bool {
	return ssacall.CallMatchesSymbol(call.Common(), syntax.Builtin("close"))
}
