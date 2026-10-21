package synchronization

import (
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// Standard mutex operations mutate their receiver's internal state; they do
// not retain a pointer to the enclosing user object after a synchronous call.
// Callers must still establish execution mode and exact storage before applying
// this contract. An interface call or RLocker adapter is not a direct call.
// https://go.dev/src/sync/mutex.go
// https://go.dev/src/sync/rwmutex.go

// MutexEffectSignature recognizes the standard receiver-mutation contract.
type MutexEffectSignature func(*ssa.CallCommon) (bool, bool)

var (
	tryLock      = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "Mutex", Name: "TryLock"})
	tryWriteLock = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "RWMutex", Name: "TryLock"})
	tryReadLock  = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "sync", Receiver: "RWMutex", Name: "TryRLock"})
)

func tryLockCall(common *ssa.CallCommon) (ssa.Value, bool) {
	return ssaflow.CallReceiver(common), ssacall.CallMatchesAnySymbol(common, tryLock, tryWriteLock, tryReadLock)
}
