package deferinloop

import (
	"strconv"

	"github.com/kojah/gohawk/internal/analysis/passes/lifecyclefacts"
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/lifecycle"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"
	analysisTrace "github.com/kojah/gohawk/internal/reporting/trace"
	"golang.org/x/tools/go/ssa"
)

// Classification is deliberately asymmetric: explicit cleanup and proven
// transfer settle, opaque use suppresses, and ordinary operations on the
// receiver leave it live. Merely failing to find a cleanup proves nothing.
// The reason names the rule that moved the status, so a trace can attribute
// a settled or unknown resource to the instruction and rule that decided it.
func classifyResourceInstruction(
	evidence *lifecyclefacts.LifecycleEvidence,
	probe analysisTrace.Probe,
	instruction ssa.Instruction,
	obligation deferObligation,
) (resourceStatus, deferReason) {
	status, reason := classifyValueUse(evidence, probe, instruction, obligation)
	if status != resourceLive || obligation.owner == nil {
		return status, reason
	}
	// The owning response is classified for escape and transfer only: none of
	// its own methods is the body's cleanup.
	return classifyValueUse(evidence, probe, instruction, deferObligation{target: obligation.owner})
}

// Transfers reuse the shared structural ownership vocabulary, including
// imported facts for callees that retain the exact resource beyond the call.
func transfersResource(
	evidence *lifecyclefacts.LifecycleEvidence,
	instruction ssa.Instruction,
	target ssa.Value,
) bool {
	if lifecycle.StoresValueInGlobal(instruction, target) ||
		lifecycle.StoresValueInEscapingField(instruction, target) ||
		lifecycle.SendsValue(instruction, target) ||
		lifecycle.CallTransfersArgumentToReturnedOwner(instruction, target) ||
		lifecycle.CallTransfersArgumentToReceiver(instruction, target) ||
		lifecycle.CallTransfersArgumentToLifecycleOwner(instruction, target) {
		return true
	}
	return evidence.ArgumentRetainedByCallee(instruction, target)
}

// A source-visible callee summarized as neither releasing nor retaining the
// argument is an ordinary use. An unreadable callee that receives the value
// makes the lifetime unknown, preserving the precision-first reporting rule.
func resourceUseStatus(
	evidence *lifecyclefacts.LifecycleEvidence,
	probe analysisTrace.Probe,
	instruction ssa.Instruction,
	target ssa.Value,
) (resourceStatus, deferReason) {
	common := ssaflow.InstructionCall(instruction)
	if common == nil || ssacall.CallMatchesAnySymbol(common, lengthBuiltins...) {
		// len and cap only read a length: a wrapper passed to them is neither
		// released nor kept. A range over a slice in the owner takes its
		// length on every iteration.
		// https://github.com/ForceCLI/force/blob/662af739b980a568fa55e3a4d7efe65cf2ec15b1/command/fetch.go#L317-L322
		return resourceLive, reasonNone
	}
	used := false
	for index, argument := range common.Args {
		alias := heapmodel.ProveMayAlias(argument, target)
		contains := !alias.Aliases && lifecycle.MayContainValue(argument, target)
		if probe.Enabled() {
			probe.Evidence(analysisTrace.Step{
				Reason: reasonArgumentCarriesResource.String(), Outcome: analysisTrace.OutcomeObserved, Pos: instruction.Pos(),
				Details: map[string]string{
					"argument": argument.Name(), "alias": strconv.FormatBool(alias.Aliases), "alias-reason": alias.Reason.String(),
					"contains": strconv.FormatBool(contains),
				},
			})
		}
		if contains {
			// The callee received a wrapper holding the resource, not the
			// resource itself. Its summary describes the argument, so it can
			// neither release nor be proven to leave the contents alone:
			// unknown, not cleanup. Constructing the wrapper alone does not
			// settle the obligation either.
			// https://github.com/replicatedhq/troubleshoot/blob/eacd376c1245fe2ebcc3581f015f692d77d89af4/pkg/supportbundle/aftercollection.go#L39-L53
			return resourceUnknown, reasonWrapperPassedToCallee
		}
		if !alias.Aliases {
			if syntax.PointerStruct(argument.Type()) != nil && heapmodel.ValueDerivesFrom(argument, target) {
				return resourceUnknown, reasonWrapperPassedToCallee
			}
			continue
		}
		used = true
		if released, summarized := evidence.CalleeClaims(instruction, index, lifecyclefacts.ClaimReleases); summarized {
			if released {
				return resourceSettled, reasonCalleeReleasesArgument
			}
		}
	}
	if used && !evidence.CalleeSummarized(instruction) {
		return resourceUnknown, reasonUnsummarizedCalleeUse
	}
	return resourceLive, reasonNone
}

// Capturing the resource in a closure or storing it in an aggregate is opaque:
// the lifetime of that containing owner is outside this loop-local proof.
func opaqueResourceUse(instruction ssa.Instruction, target ssa.Value) bool {
	if store, ok := instruction.(*ssa.Store); ok {
		switch store.Addr.(type) {
		case *ssa.FieldAddr, *ssa.IndexAddr:
			// A local aggregate whose address never leaves the function
			// lives no longer than this iteration, so a resource stored in
			// it, and closed through it, is still iteration-local.
			// Result edges end execution rather than reaching the next
			// iteration. Ask about body retention, not ownership at return.
			if heapmodel.QueryEscape(store.Addr, heapmodel.EscapeBody).Outcome == heapmodel.EscapeLocal {
				return false
			}
			return heapmodel.MayAlias(store.Val, target) || lifecycle.MayContainValue(store.Val, target)
		}
	}
	closure, ok := instruction.(*ssa.MakeClosure)
	if !ok {
		return false
	}
	for _, binding := range closure.Bindings {
		if heapmodel.MayAlias(binding, target) || lifecycle.MayContainValue(binding, target) {
			return true
		}
	}
	return false
}

var lengthBuiltins = []syntax.Symbol{syntax.Builtin("len"), syntax.Builtin("cap")}
