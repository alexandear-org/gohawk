package httpacquire

import (
	"go/constant"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"

	"golang.org/x/tools/go/ssa"
)

// A HEAD response through an unconfigured client normally carries
// http.NoBody, not an acquired body: the standard transport reads no body for
// HEAD, and without Client.Timeout there is no cancelTimerBody around it.
// DefaultTransport and DefaultClient are replaceable, so this is uncertainty
// about acquisition, never a proof that closing is unnecessary. The client is
// either a fresh zero-value local used only by Do, or the package default
// client with no visible reconfiguration in this function or a visible callee;
// hidden cross-package mutation of the defaults is the same accepted coverage
// gap as the local-server boundary below. The request must be the direct
// HEAD constructor result, possibly rebound through WithContext or Clone and
// with its Header map edited, none of which can change Method. Any other use
// of the request or the client, such as a helper receiving it, could.
// https://github.com/vishen/go-chromecast/blob/5dd70bb91787fe28e3d8946682c66cb2a1d61d21/application/application.go#L723-L732
// https://github.com/alexellis/arkade/blob/0a0a800fd7554d4eddb1856f9ef8a21214e95bab/pkg/get/get.go#L236-L244
// https://github.com/deweizhu/bookget/blob/2cdbf6d6c3ce70355a5c4411c0faf3450e9ae877/pkg/downloader/downloader.go#L510-L522

// Head retains the final caller/pool fence after the ordered evidence queries.
func Head(call *ssa.Call, budget *proofs.SearchBudget) Proof {
	return decideHead(call, budget).within(budget)
}

type headSignature func(*ssa.Call, *proofs.SearchBudget) Proof

var _ headSignature = decideHead

func headDo(call *ssa.Call) (ssa.Value, ssa.Value, bool) {
	common := call.Common()
	if !ssacall.CallMatchesSymbol(common, ClientDo) || len(common.Args) != 2 {
		return nil, nil, false
	}
	return common.Args[0], common.Args[1], true
}

func headClient(client ssa.Value, call *ssa.Call, budget *proofs.SearchBudget) Proof {
	return proveHeadClientUnconfiguredWithin(client, call.Parent(), budget)
}

// ClientDo names the exact standard client request contract.
var (
	ClientDo               = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http", Receiver: "Client", Name: "Do"})
	httpDefaultClient      = syntax.PackageVariable("net/http", "DefaultClient")
	httpDefaultTransport   = syntax.PackageVariable("net/http", "DefaultTransport")
	httpRequestWithContext = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http", Receiver: "Request", Name: "WithContext"})
	httpRequestClone       = syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http", Receiver: "Request", Name: "Clone"})
	// Header edits through the standard map methods cannot change the request
	// method; the map handed anywhere else is rejected so the rule stays small.
	httpHeaderEdits = []syntax.Symbol{
		syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http", Receiver: "Header", Name: "Set"}),
		syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http", Receiver: "Header", Name: "Add"}),
		syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http", Receiver: "Header", Name: "Get"}),
		syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http", Receiver: "Header", Name: "Del"}),
		syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http", Receiver: "Header", Name: "Values"}),
	}
)

func constantString(value ssa.Value) string {
	text, ok := value.(*ssa.Const)
	if !ok || text.Value == nil || text.Value.Kind() != constant.String {
		return ""
	}
	return constant.StringVal(text.Value)
}
