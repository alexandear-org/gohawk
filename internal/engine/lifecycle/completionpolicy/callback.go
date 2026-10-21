package completionpolicy

import (
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// CallbackKind selects one exact closure, one transparent callback source or
// a storage query. Missing storage evidence cannot establish callback identity.
type CallbackKind uint8

const (
	CallbackAbsent CallbackKind = iota
	CallbackExact
	CallbackFollow
	CallbackStorage
)

// CallbackSource supplies one step without traversing wrappers, storage or phis.
type CallbackSource struct {
	Kind    CallbackKind
	Closure *ssa.MakeClosure
	Source  ssa.Value
}
type callbackSignature func(ssa.Value, bool) CallbackSource

var _ callbackSignature = RecognizeCallback

var onceFunc = syntax.PackageFunction("sync", "OnceFunc")

func onceSource(common ssa.CallCommon) (ssa.Value, bool) {
	if !ssacall.CallMatchesSymbol(&common, onceFunc) || len(common.Args) != 1 {
		return nil, false
	}
	return common.Args[0], true
}
