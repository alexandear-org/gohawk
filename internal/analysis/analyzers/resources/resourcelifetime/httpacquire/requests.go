package httpacquire

import (
	"go/token"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// HEAD clients must be exact fresh zero-value allocations, stable local cells,
// or unchanged package defaults. Referrer and capture visits share the request
// allowance; shortened default-effect children retain explicit uncertainty.

func proveHeadClientUnconfiguredWithin(client ssa.Value, function *ssa.Function, budget *proofs.SearchBudget) Proof {
	if !budget.Spend() {
		return carriedProof(false, budget)
	}
	return decideHeadClient(client, function, budget)
}

// zeroClientCell accepts a local variable that holds one fresh zero-value
// client and is only ever loaded for direct Do calls, here or inside a
// literal that captures it. A worker that shares the client can still
// change nothing about it. pmtiles keeps one client in a variable its
// download workers capture:
// https://github.com/protomaps/go-pmtiles/blob/a3e4951ea6a0477b784c27c1dcbfd9c130878c5a/pmtiles/sync.go#L73-L80
func zeroClientCellWithin(cell *ssa.Alloc, budget *proofs.SearchBudget) bool {
	if cell.Referrers() == nil {
		return false
	}
	stored := false
	for _, use := range *cell.Referrers() {
		if !budget.Spend() {
			return false
		}
		accepted, next := decideClientCellUse(use, cell, stored, budget)
		if !accepted {
			return false
		}
		stored = next
	}
	return stored
}

// onlyStoredInto reports whether the fresh allocation's only use is the store
// that puts it in the cell: no field was addressed, so it is zero-valued.
func onlyStoredIntoWithin(fresh *ssa.Alloc, store *ssa.Store, budget *proofs.SearchBudget) bool {
	if fresh.Referrers() == nil {
		return false
	}
	for _, use := range *fresh.Referrers() {
		if !budget.Spend() {
			return false
		}
		if _, debug := use.(*ssa.DebugRef); !debug && use != store {
			return false
		}
	}
	return true
}

func closureLoadsCellForDoWithin(closure *ssa.MakeClosure, cell *ssa.Alloc, budget *proofs.SearchBudget) bool {
	function, ok := closure.Fn.(*ssa.Function)
	if !ok {
		return false
	}
	for _, pair := range ssaflow.ClosureBindingPairs(function, closure) {
		if !budget.Spend() {
			return false
		}
		if pair.Binding != cell {
			continue
		}
		free := pair.Free
		if free.Referrers() == nil {
			return false
		}
		for _, use := range *free.Referrers() {
			if !budget.Spend() {
				return false
			}
			load, loaded := use.(*ssa.UnOp)
			if _, debug := use.(*ssa.DebugRef); debug {
				continue
			}
			if !loaded || load.Op != token.MUL || !onlyHTTPDoUsesWithin(load, budget) {
				return false
			}
		}
	}
	return true
}

// Only direct Do calls may observe these fresh values. Field addresses, aliases
// and helper escapes would hide configuration or method mutation.
func onlyHTTPDoUsesWithin(value ssa.Value, budget *proofs.SearchBudget) bool {
	refs := value.Referrers()
	if refs == nil || len(*refs) == 0 {
		return false
	}
	for _, ref := range *refs {
		if !budget.Spend() {
			return false
		}
		call, ok := ref.(*ssa.Call)
		if !ok || !ssacall.CallMatchesSymbol(call.Common(), ClientDo) {
			return false
		}
	}
	return true
}

// HEAD provenance follows only exact constructor results and rebinding calls.
// Backward origin and forward immutable-use folds share mechanics and allowance;
// phis, aliases and unmodeled uses remain outside the contract.

// HEAD constructor and immutable-use queries share exact rebinding mechanics.
// Only call and extract roots are transparent; phis remain outside this policy.
func headConstructedWithin(request ssa.Value, budget *proofs.SearchBudget) bool {
	return (headRequestQuery{budget: budget}).prove(request)
}

type headRequestQuery struct {
	budget       *proofs.SearchBudget
	preserveUses bool
}

func headRequestWithin(request ssa.Value, budget *proofs.SearchBudget) bool {
	return (headRequestQuery{budget: budget, preserveUses: true}).prove(request)
}

func directHEADRequestValue(value ssa.Value) bool {
	switch value.(type) {
	case *ssa.Call, *ssa.Extract:
		return true
	}
	return false
}

// Preserve the former direct-chain boundary before entering the shared fold:
// phi fan-out could otherwise broaden a HEAD contract to merged requests.
func (query headRequestQuery) prove(request ssa.Value) bool {
	return directHEADRequestValue(request) && ssaflow.NewReachingWalk(ssaflow.TransparentNone).Within(query.budget).Every(request, query.origin)
}

func (query headRequestQuery) origin(walk ssaflow.ReachingWalk, value ssa.Value) bool {
	return decideRequestOrigin(query, walk, value)
}

func (query headRequestQuery) followOrigin(walk ssaflow.ReachingWalk, value ssa.Value) bool {
	return walk.Every(value, query.origin)
}

func (query headRequestQuery) preserve(value ssa.Value) bool {
	return !query.preserveUses || ssaflow.NewReachingWalk(ssaflow.TransparentNone).Within(query.budget).Every(value, query.uses)
}

func headConstructor(common *ssa.CallCommon) bool {
	if ssacall.CallMatchesSymbol(common, syntax.PackageFunction("net/http", "NewRequest")) {
		return len(common.Args) == 3 && constantString(common.Args[0]) == "HEAD"
	}
	if ssacall.CallMatchesSymbol(common, syntax.PackageFunction("net/http", "NewRequestWithContext")) {
		return len(common.Args) == 4 && constantString(common.Args[1]) == "HEAD"
	}
	return false
}

// Origins and uses are separate folds. A backward chain visits the receiver,
// while this forward query must independently check every derived request for
// mutation or escape. Reusing the origin fold's visited set would discard valid
// rebinding evidence; sharing the allowance still bounds both directions.
func (query headRequestQuery) uses(walk ssaflow.ReachingWalk, request ssa.Value) bool {
	refs := request.Referrers()
	if refs == nil || len(*refs) == 0 {
		return false
	}
	for _, ref := range *refs {
		if !query.budget.Spend() || !decideRequestUse(query, walk, request, ref) {
			return false
		}
	}
	return true
}

func (query headRequestQuery) followUses(walk ssaflow.ReachingWalk, request ssa.Value) bool {
	return walk.Every(request, query.uses)
}

func requestFieldName(field *ssa.FieldAddr) string {
	structure := syntax.PointerStruct(field.X.Type())
	if structure == nil || !syntax.NamedType(field.X.Type(), "net/http", "Request") {
		return ""
	}
	return structure.Field(field.Field).Name()
}

// Only known header-map operations preserve the request contract. Sending the
// map to opaque code is not a general purity proof, even if Method is untouched
// by every use recognized so far; shortened use scans remain unavailable.
func headerUsesAreEditsWithin(field *ssa.FieldAddr, budget *proofs.SearchBudget) bool {
	if field.Referrers() == nil {
		return false
	}
	for _, use := range *field.Referrers() {
		if !budget.Spend() {
			return false
		}
		load, ok := use.(*ssa.UnOp)
		if !ok || load.Op != token.MUL || load.Referrers() == nil {
			return false
		}
		for _, edit := range *load.Referrers() {
			if !budget.Spend() {
				return false
			}
			call, ok := edit.(*ssa.Call)
			if !ok || ssaflow.CallReceiver(call.Common()) != load || !ssacall.CallMatchesAnySymbol(call.Common(), httpHeaderEdits...) {
				return false
			}
		}
	}
	return true
}

// Visible default-client and transport effects share one scan for HEAD and
// local header-only acquisitions. Only HEAD's root permits a default-client
// load used exclusively by Do; that allowance never enters helper summaries.
// Only visible bodies are expanded; recursive or shortened evidence cannot
// establish absence of visible changes. Hidden package effects stay opaque.

// EffectsBudget bounds visible-body effect queries. Both HTTP acquisition
// families retain this quota; exhaustion supplies the conservative may-modify
// answer rather than evidence that a global default is unchanged.
const EffectsBudget = 4000

// DefaultClientUnmodified keeps its existing child cap while sharing the caller's
// allowance. A shortened child is unavailable even when the caller can continue.
// DefaultClientUnmodified permits only the root's direct Do uses of the default client.
func DefaultClientUnmodified(function *ssa.Function, budget *proofs.SearchBudget) Proof {
	child := budget.Within(EffectsBudget)
	modified := NewEffects().ScanOverrides(function, child, true)
	return carriedProof(!modified, child)
}

func (effects *Effects) visibleOverrides(function *ssa.Function, budget *proofs.SearchBudget) bool {
	return effects.ScanOverrides(function, budget, false)
}

// ScanOverrides inspects visible defaults, allowing root Do reads only when selected.
// Nested helper summaries always retain the strict policy.
func (effects *Effects) ScanOverrides(function *ssa.Function, budget *proofs.SearchBudget, allowRootDo bool) bool {
	for instruction := range ssaflow.InstructionsWithin(function, budget) {
		if decideDefaultUse(effects, instruction, allowRootDo, budget) {
			return true
		}
	}
	return exhausted(budget)
}

func defaultOperands(instruction ssa.Instruction, budget *proofs.SearchBudget) bool {
	for _, operand := range instruction.Operands(nil) {
		if !budget.Spend() {
			return true
		}
		if operand != nil && ssacall.ValueMatchesAnySymbol(*operand, httpDefaultClient, httpDefaultTransport) {
			return true
		}
	}
	return false
}

func (effects *Effects) helperOverrides(instruction ssa.Instruction, budget *proofs.SearchBudget) bool {
	callee, _ := ssacall.DirectCallee(ssaflow.InstructionCall(instruction))
	return callee != nil && len(callee.Blocks) != 0 && effects.overrides.Function(callee, budget)
}
