package httpacquire

import (
	"go/token"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// These bindings expose exact HTTP use forms to the generated policies. The
// surrounding scans and reaching folds own every visit, recursive query and
// cutoff; recognition never establishes unchanged state from missing evidence.
type (
	headClientSignature    func(ssa.Value, *ssa.Function, *proofs.SearchBudget) Proof
	clientCellUseSignature func(ssa.Instruction, *ssa.Alloc, bool, *proofs.SearchBudget) (bool, bool)
	requestOriginSignature func(headRequestQuery, ssaflow.ReachingWalk, ssa.Value) bool
	requestUseSignature    func(headRequestQuery, ssaflow.ReachingWalk, ssa.Value, ssa.Instruction) bool
	serverUseSignature     func(ssa.Instruction, *proofs.SearchBudget) bool
	writerUseSignature     func(*Effects, ssa.Instruction, ssa.Value, *proofs.SearchBudget) bool
	defaultUseSignature    func(*Effects, ssa.Instruction, bool, *proofs.SearchBudget) bool
)

var (
	_ headClientSignature    = decideHeadClient
	_ clientCellUseSignature = decideClientCellUse
	_ requestOriginSignature = decideRequestOrigin
	_ requestUseSignature    = decideRequestUse
	_ serverUseSignature     = decideServerUse
	_ writerUseSignature     = decideWriterUse
	_ defaultUseSignature    = decideDefaultUse
)

const loadToken = token.MUL

var (
	setCookieSymbol    = syntax.PackageFunction("net/http", "SetCookie")
	writerHeaderSymbol = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http", Receiver: "ResponseWriter", Name: "Header"})
	writerStatusSymbol = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http", Receiver: "ResponseWriter", Name: "WriteHeader"})
	serverCloseSymbol  = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http/httptest", Receiver: "Server", Name: "Close"})
)

func cookieCall(call *ssa.Call) bool {
	return ssacall.CallMatchesSymbol(call.Common(), setCookieSymbol)
}

func writerHeaderCall(call *ssa.Call) bool {
	return ssacall.CallMatchesSymbol(call.Common(), writerHeaderSymbol)
}

func writerStatusCall(call *ssa.Call) bool {
	return ssacall.CallMatchesSymbol(call.Common(), writerStatusSymbol)
}

func serverClientCall(call *ssa.Call) bool {
	return ssacall.CallMatchesSymbol(call.Common(), httptestServerClient)
}

func defaultClientValue(value ssa.Value) bool {
	return ssacall.ValueMatchesSymbol(value, httpDefaultClient)
}

func sameValue(first, second ssa.Value) bool { return first == second }

func constructorCall(call *ssa.Call) bool { return headConstructor(call.Common()) }

func requestRebinding(common ssa.CallCommon) (ssa.Value, bool) {
	if !ssacall.CallMatchesAnySymbol(&common, httpRequestWithContext, httpRequestClone) {
		return nil, false
	}
	receiver := ssaflow.CallReceiver(&common)
	return receiver, directHEADRequestValue(receiver)
}

func requestDo(call *ssa.Call, request ssa.Value) bool {
	common := call.Common()
	return ssacall.CallMatchesSymbol(common, ClientDo) && len(common.Args) == 2 && common.Args[1] == request
}

func requestRebindUse(call *ssa.Call, request ssa.Value) bool {
	common := call.Common()
	return ssacall.CallMatchesAnySymbol(common, httpRequestWithContext, httpRequestClone) && ssaflow.CallReceiver(common) == request
}

func requestHeader(field *ssa.FieldAddr) bool { return requestFieldName(field) == "Header" }

func hasReferrers(value ssa.Value) bool { return value.Referrers() != nil }

func serverClose(instruction ssa.Instruction) bool {
	return ssacall.CallMatchesSymbol(ssaflow.InstructionCall(instruction), serverCloseSymbol)
}
