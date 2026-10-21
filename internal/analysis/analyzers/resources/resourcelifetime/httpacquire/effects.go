package httpacquire

import (
	"go/constant"
	"strings"

	"github.com/kojah/gohawk/internal/engine/heapmodel"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"

	"golang.org/x/tools/go/ssa"
)

// Header-only effects require every visible consumption of the exact writer
// to preserve empty response framing. Opaque, recursive or shortened helper
// evidence cannot prove this contract; alias internals retain separate costs.

// Effects owns independent bounded writer and default-mutation memos.
// Interrupted or recursive queries retain their conservative fallback.
type Effects struct {
	writers   *ssacall.CallGraphMemo[*ssa.Parameter, bool]
	overrides *ssacall.FunctionSummaries[bool]
}

// NewEffects creates independent memoized writer/default-effect searches.
func NewEffects() *Effects {
	effects := &Effects{writers: ssacall.NewCallGraphMemo[*ssa.Parameter, bool]()}
	effects.overrides = ssacall.NewFunctionSummaries(effects.visibleOverrides, func(ssacall.SummaryUnavailable) bool { return true })
	return effects
}

// HeaderOnly requires every visible use of the writer to preserve empty framing.
func (effects *Effects) HeaderOnly(writer *ssa.Parameter, budget *proofs.SearchBudget) bool {
	return effects.writers.Summarize(writer, writer.Parent(), budget, func() bool {
		for instruction := range ssaflow.InstructionsWithin(writer.Parent(), budget) {
			for _, operand := range instruction.Operands(nil) {
				if !budget.Spend() {
					return false
				}
				if operand != nil && budget.Spend() && heapmodel.MayAlias(*operand, writer) && !effects.writerUse(instruction, writer, budget) {
					return false
				}
			}
		}
		return !exhausted(budget)
	}, func(ssacall.SummaryUnavailable, bool) bool { return false })
}

func (effects *Effects) writerUse(instruction ssa.Instruction, writer ssa.Value, budget *proofs.SearchBudget) bool {
	if _, ok := instruction.(*ssa.ChangeInterface); ok {
		return true
	}
	call, ok := instruction.(*ssa.Call)
	if !ok {
		return false
	}
	common := call.Common()
	if ssacall.CallMatchesSymbol(common, syntax.PackageFunction("net/http", "SetCookie")) {
		return true
	}
	if ssacall.CallMatchesSymbol(common, syntax.PackageMethod(syntax.MethodSymbol{
		PackagePath: "net/http", Receiver: "ResponseWriter", Name: "Header",
	})) {
		return benignHTTPHeaders(call, budget)
	}
	if ssacall.CallMatchesSymbol(common, syntax.PackageMethod(syntax.MethodSymbol{
		PackagePath: "net/http", Receiver: "ResponseWriter", Name: "WriteHeader",
	})) {
		code, ok := common.Args[0].(*ssa.Const)
		if !ok || code.Value == nil || code.Value.Kind() != constant.Int {
			return false
		}
		n, exact := constant.Int64Val(code.Value)
		return exact && n >= 200 && n <= 599 && (n < 300 || n >= 400)
	}
	callee, closure := ssacall.DirectCallee(common)
	// A visible helper must preserve the same restriction for every parameter
	// receiving the writer. Missing bodies or captured/dynamic dispatch are not
	// evidence of an empty effect set, so they cannot establish empty framing.
	found := false
	for _, binding := range ssacall.CallBindings(common, callee, closure) {
		if !budget.Spend() {
			return false
		}
		if !budget.Spend() {
			return false
		}
		if !heapmodel.MayAlias(binding.Supplied, writer) {
			continue
		}
		parameter, ok := binding.Local.(*ssa.Parameter)
		if !ok || !effects.HeaderOnly(parameter, budget) {
			return false
		}
		found = true
	}
	return found
}

func benignHTTPHeaders(header *ssa.Call, budget *proofs.SearchBudget) bool {
	if header.Referrers() == nil {
		return false
	}
	for _, ref := range *header.Referrers() {
		if !budget.Spend() {
			return false
		}
		call, ok := ref.(*ssa.Call)
		if !ok || len(call.Common().Args) < 2 {
			return false
		}
		if !ssacall.CallMatchesAnySymbol(call.Common(),
			syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http", Receiver: "Header", Name: "Set"}),
			syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "net/http", Receiver: "Header", Name: "Add"})) {
			return false
		}
		key := strings.ToLower(constantString(call.Common().Args[1]))
		switch key {
		case "", "content-length", "transfer-encoding", "trailer", "connection", "upgrade", "location":
			return false
		}
	}
	return true
}

// Overrides asks the strict declaration summary. Root-only allowances cannot
// enter this memo or replace its unknown/recursive default.
func (effects *Effects) Overrides(function *ssa.Function, budget *proofs.SearchBudget) bool {
	return effects.overrides.Function(function, budget)
}

type effectsSignature func(*Effects, *ssa.Function, *ssa.Function, *proofs.SearchBudget) Proof

var _ effectsSignature = decideEffects

func handlerHeaders(effects *Effects, function *ssa.Function, budget *proofs.SearchBudget) bool {
	return effects.HeaderOnly(function.Params[0], budget)
}
