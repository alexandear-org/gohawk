package synchronization

import (
	"go/constant"

	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// Primitive carries a recognized operation's original receiver and an Add delta
// when it is a known integer. Consumers decide which counter changes they model;
// recognizing a call establishes neither identity nor lifecycle completion.
type Primitive struct {
	Kind     Kind
	Resource ssa.Value
	Fixed    bool
	Count    int64
}

// PrimitiveSignature describes standard-library operation recognition.
type PrimitiveSignature func(*ssa.CallCommon) (Primitive, bool)

// Exact declaration identity keeps lookalike project methods outside this model.
// Read and write modes remain distinct even when they name the same resource.
var (
	groupAdd     = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "WaitGroup", Name: "Add"})
	groupDone    = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "WaitGroup", Name: "Done"})
	groupWait    = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "WaitGroup", Name: "Wait"})
	mutexLock    = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "Mutex", Name: "Lock"})
	mutexUnlock  = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "Mutex", Name: "Unlock"})
	rwLock       = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "RWMutex", Name: "Lock"})
	rwUnlock     = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "RWMutex", Name: "Unlock"})
	rwReadLock   = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "RWMutex", Name: "RLock"})
	rwReadUnlock = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "RWMutex", Name: "RUnlock"})
	condWait     = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "Cond", Name: "Wait"})
)

func lockCall(common *ssa.CallCommon) (ssa.Value, bool) {
	return ssaflow.CallReceiver(common), ssacall.CallMatchesAnySymbol(common, mutexLock, rwLock)
}

func unlockCall(common *ssa.CallCommon) (ssa.Value, bool) {
	return ssaflow.CallReceiver(common), ssacall.CallMatchesAnySymbol(common, mutexUnlock, rwUnlock)
}

func readLockCall(common *ssa.CallCommon) (ssa.Value, bool) {
	return ssaflow.CallReceiver(common), ssacall.CallMatchesSymbol(common, rwReadLock)
}

func readUnlockCall(common *ssa.CallCommon) (ssa.Value, bool) {
	return ssaflow.CallReceiver(common), ssacall.CallMatchesSymbol(common, rwReadUnlock)
}

func waitCall(common *ssa.CallCommon) (ssa.Value, bool) {
	return ssaflow.CallReceiver(common), ssacall.CallMatchesSymbol(common, groupWait)
}

func doneCall(common *ssa.CallCommon) (ssa.Value, bool) {
	return ssaflow.CallReceiver(common), ssacall.CallMatchesSymbol(common, groupDone)
}

func condCall(common *ssa.CallCommon) (ssa.Value, bool) {
	return ssaflow.CallReceiver(common), ssacall.CallMatchesSymbol(common, condWait)
}

func closeCall(common *ssa.CallCommon) (ssa.Value, bool) {
	if !ssacall.CallMatchesSymbol(common, syntax.Builtin("close")) {
		return nil, false
	}
	return common.Args[0], true
}

func addCall(common *ssa.CallCommon) (receiver, delta ssa.Value, matched bool) {
	if !ssacall.CallMatchesSymbol(common, groupAdd) {
		return nil, nil, false
	}
	if len(common.Args) != 2 {
		return ssaflow.CallReceiver(common), nil, true
	}
	return ssaflow.CallReceiver(common), common.Args[1], true
}

func integer(value ssa.Value) (int64, bool) {
	count, ok := value.(*ssa.Const)
	if !ok || count.Value == nil || count.Value.Kind() != constant.Int {
		return 0, false
	}
	return constant.Int64Val(count.Value)
}
