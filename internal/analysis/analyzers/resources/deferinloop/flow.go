package deferinloop

import (
	"slices"
	"strconv"

	"github.com/kojah/gohawk/internal/analysis/passes/lifecyclefacts"
	"github.com/kojah/gohawk/internal/analysis/summaries"
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	analysisTrace "github.com/kojah/gohawk/internal/reporting/trace"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	cfg "github.com/kojah/gohawk/internal/engine/ssaflow/cfg"
	ssapath "github.com/kojah/gohawk/internal/engine/ssaflow/path"
	"golang.org/x/tools/go/ssa"
)

type resourceStatus uint8

const (
	resourceLive resourceStatus = iota
	resourceSettled
	resourceUnknown
)

type deferFlowState struct {
	block       *ssa.BasicBlock
	predecessor *ssa.BasicBlock
	index       int
	status      resourceStatus
}

// deferLifetimeProof describes the live-backedge reporting question. Proven
// means a definitely live resource reaches the next iteration; unknown never
// claims cleanup. The witness retains the selected evidence position.
type deferLifetimeProof struct {
	state    proofs.EvidenceState
	reason   deferReason
	witness  ssa.Instruction
	backedge *ssa.BasicBlock
}

// proveDeferLifetime asks the narrow reporting question directly:
// can this exact acquired value reach a block dominating the defer's block
// while still definitely live? Returning is safe because Go runs the defer;
// settled and unknown paths stop at the backedge without producing a claim.
func proveDeferLifetime(
	evidence *lifecyclefacts.LifecycleEvidence,
	knowledge *summaries.Provider,
	probe analysisTrace.Probe,
	deferred *ssa.Defer,
	obligation deferObligation,
) deferLifetimeProof {
	index := cfg.InstructionIndex(deferred)
	if index < 0 {
		return deferLifetimeProof{state: proofs.EvidenceUnknown, reason: reasonDeferLocationUnknown, witness: deferred}
	}
	// A resource retained before its defer is no more iteration-local than one
	// retained afterward. In particular append lowers to an indexed store
	// before the append call; the containing collection may be consumed later.
	// https://github.com/protomaps/go-pmtiles/blob/a3e4951ea6a0477b784c27c1dcbfd9c130878c5a/pmtiles/merge.go#L206-L215
	for _, store := range ssaflow.InstructionsOf[*ssa.Store](deferred.Parent()) {
		retains := func(value ssa.Value) bool { return opaqueResourceUse(store, value) }
		if cfg.InstructionDominates(store, deferred) && slices.ContainsFunc(obligation.values(), retains) {
			return deferLifetimeProof{state: proofs.EvidenceUnknown, reason: reasonRetainedBeforeDefer, witness: store}
		}
	}
	proof := deferLifetimeProof{state: proofs.EvidenceDisproven, reason: reasonNoLiveBackedge, witness: deferred}
	unknownAtBackedge := false
	initial := []deferFlowState{{block: deferred.Block(), index: index + 1, status: resourceLive}}
	cfg.WalkStates(initial, func(state deferFlowState) deferFlowState { return state }, func(state deferFlowState) ([]deferFlowState, bool) {
		state = advanceDeferState(evidence, probe, state, obligation)
		// A branch a callee's proven result rules out is not a path to the
		// backedge; feasibility only removes successors, it never adds one.
		feasible := ssapath.SuccessorPolicy{Feasible: func(block, predecessor *ssa.BasicBlock) []*ssa.BasicBlock {
			return knowledge.FeasibleSuccessors(block, predecessor, proofs.NewSearchBudget(proofs.SummaryBudget))
		}}.Successors(state.block, state.predecessor)
		successors := make([]deferFlowState, 0, len(feasible))
		for _, successor := range feasible {
			status := iteratorSuccessorStatus(state, successor, obligation)
			if successor.Dominates(deferred.Block()) {
				if status == resourceLive {
					proof = deferLifetimeProof{
						state: proofs.EvidenceProven, reason: reasonLiveAtBackedge, witness: deferred, backedge: state.block,
					}
					return nil, false
				}
				unknownAtBackedge = unknownAtBackedge || status == resourceUnknown
				if status != state.status && probe.Enabled() {
					probe.Evidence(analysisTrace.Step{
						Reason: reasonIteratorExhausted.String(), Outcome: analysisTrace.OutcomeUnknown, Pos: deferred.Pos(),
						Details: map[string]string{"block": strconv.Itoa(state.block.Index)},
					})
				}
				continue
			}
			successors = append(successors, deferFlowState{
				block: successor, predecessor: state.block, status: status,
			})
		}
		return successors, true
	})
	// An opaque path cannot erase a separate definitely-live witness. Only
	// when no live path wins does unknown backedge evidence suppress the proof.
	if proof.state != proofs.EvidenceProven && unknownAtBackedge {
		proof.state, proof.reason = proofs.EvidenceUnknown, reasonLifetimeUnknownAtBackedge
	}
	return proof
}

// Without a general contract proving what Next does on exhaustion, its false
// branch leaves the lifetime unknown and suppresses a claim. The true branch
// stays live, so an early break can still reach the outer backedge and report.
func iteratorSuccessorStatus(
	state deferFlowState,
	successor *ssa.BasicBlock,
	obligation deferObligation,
) resourceStatus {
	if state.status != resourceLive || len(state.block.Succs) != 2 || successor != state.block.Succs[1] {
		return state.status
	}
	branch, ok := state.block.Instrs[len(state.block.Instrs)-1].(*ssa.If)
	if !ok {
		return state.status
	}
	iterator := conditionCall(state.block, branch.Cond)
	if iterator == nil || ssaflow.CallName(iterator.Common()) != "Next" ||
		!sameObligationValue(ssaflow.CallReceiver(iterator.Common()), obligation.target) {
		return state.status
	}
	return resourceUnknown
}

func conditionCall(block *ssa.BasicBlock, condition ssa.Value) *ssa.Call {
	for _, instruction := range block.Instrs {
		call, ok := instruction.(*ssa.Call)
		if ok && heapmodel.MayAlias(condition, call) {
			return call
		}
	}
	return nil
}

// Each instruction advances a monotone three-state obligation. Once evidence
// settles the value, or an opaque boundary makes its lifetime unknowable,
// later instructions cannot restore the proof that it is definitely live.
func advanceDeferState(
	evidence *lifecyclefacts.LifecycleEvidence,
	probe analysisTrace.Probe,
	state deferFlowState,
	obligation deferObligation,
) deferFlowState {
	for ; state.index < len(state.block.Instrs); state.index++ {
		if state.status != resourceLive {
			// The suffix has no classification or charges once this path is
			// settled or unknown. Preserve its final position without scanning.
			state.index = len(state.block.Instrs)
			return state
		}
		instruction := state.block.Instrs[state.index]
		status, reason := classifyResourceInstruction(evidence, probe, instruction, obligation)
		if status == resourceLive {
			continue
		}
		state.status = status
		if probe.Enabled() {
			outcome := analysisTrace.OutcomeAccepted
			if status == resourceUnknown {
				outcome = analysisTrace.OutcomeUnknown
			}
			probe.Evidence(analysisTrace.Step{
				Reason: reason.String(), Outcome: outcome, Pos: instruction.Pos(),
				Details: map[string]string{"instruction": instruction.String()},
			})
		}
	}
	return state
}
