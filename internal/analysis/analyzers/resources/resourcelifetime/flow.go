package resourcelifetime

import (
	"slices"
	"strconv"

	"github.com/kojah/gohawk/internal/analysis/analyzers/resources/resourcelifetime/resourcepolicy"
	"github.com/kojah/gohawk/internal/analysis/passes/lifecyclefacts"
	"github.com/kojah/gohawk/internal/engine/lifecycle"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/resourcemodel"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	cfg "github.com/kojah/gohawk/internal/engine/ssaflow/cfg"
	ssapath "github.com/kojah/gohawk/internal/engine/ssaflow/path"
	"github.com/kojah/gohawk/internal/reporting/check"
	analysisTrace "github.com/kojah/gohawk/internal/reporting/trace"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ssa"
)

// Resource flow owns candidate initialization and the path work list over
// activation, release and uncertainty. Classifiers supply labels; this engine
// uses one state model to determine which normal returns remain uncovered.

// Resource flow tracks an acquired value from the successful call edge to each
// feasible normal return. State records activation and release separately so
// error-only resources and path-specific cleanup do not create false leaks.

// evaluateResourceFlow collects acquisition evidence, asks the path proof and
// applies the bounded policy exclusions to its diagnostic witness.
func evaluateResourceFlow(
	pass *analysis.Pass,
	evidence *lifecyclefacts.LifecycleEvidence,
	call *ssa.Call,
	resource ssa.Value,
	contract resourceContract,
) resourceLifetimePolicyResult {
	decision := resourcepolicy.Evaluate(evaluationQueries{pass: pass, evidence: evidence, call: call, resource: resource, contract: contract})
	return resourceLifetimePolicyResult{state: decision.State, reason: decision.Reason, leak: decision.Leak}
}

// emitAction traces a settled or unknown label. An instruction labelled none
// is not traced: every instruction after the acquisition would emit one, and
// the flow's return-path steps already show where the obligation stood.
func (analysis *resourceAnalysis) emitAction(instruction ssa.Instruction, action resourceAction, reason resourceLifetimeReason) {
	if action == actionNone || !analysis.probe.Enabled() {
		return
	}
	label, outcome := "settled", analysisTrace.OutcomeAccepted
	if action == actionUnknown {
		label, outcome = "unknown", analysisTrace.OutcomeUnknown
	}
	analysis.probe.Label(analysisTrace.Step{
		Reason:   reason.String(),
		Outcome:  outcome,
		Pos:      instruction.Pos(),
		Function: analysis.function.String(),
		Details:  map[string]string{"instruction": instruction.String(), "label": label},
	})
}

// localCollection finds the local slice the resource was appended to, and
// traces the decision: the collection, or the use of it that declined the
// model and left the append unknown.
func (analysis *resourceAnalysis) localCollection() *localCollection {
	decision := findLocalCollection(analysis.evidence, analysis.resource, analysis.contract.cleanup, analysis.budget(proofs.QueryBudget))
	if !analysis.probe.Enabled() || decision.collection == nil && decision.declinedAt == nil {
		return decision.collection
	}
	step := analysisTrace.Step{
		Reason: resourceReasonAppendedToLocalCollection.String(), Outcome: analysisTrace.OutcomeAccepted,
		Function: analysis.function.String(), Details: map[string]string{},
	}
	if decision.collection == nil {
		step.Reason, step.Outcome, step.Pos = resourceReasonCollectionUseUnknown.String(), analysisTrace.OutcomeUnknown, decision.declinedAt.Pos()
		step.Details["instruction"] = decision.declinedAt.String()
	} else {
		step.Pos = decision.collection.appends[0].Pos()
		step.Details["loops"] = strconv.Itoa(len(decision.collection.released))
	}
	analysis.probe.Evidence(step)
	return decision.collection
}

// traceCollectionReleased records that the path left a loop that released
// every element of the collection holding the resource.
func (analysis *resourceAnalysis) traceCollectionReleased(header, done *ssa.BasicBlock) {
	if !analysis.probe.Enabled() {
		return
	}
	test := header.Instrs[len(header.Instrs)-1]
	analysis.probe.Evidence(analysisTrace.Step{
		Reason: resourceReasonCollectionReleased.String(), Outcome: analysisTrace.OutcomeAccepted, Pos: test.Pos(),
		Function: analysis.function.String(), Details: map[string]string{"loop": test.String(), "done": strconv.Itoa(done.Index)},
	})
}

// traceUncertainEdge records an edge that made the obligation unknown: one
// that re-tests a guard the path already took the other way, or the false
// edge of Rows.Next, which may or may not have closed the rows. It is traced
// as a label because, like an opaque instruction, it is where the proof gave
// up on this path.
func (analysis *resourceAnalysis) traceUncertainEdge(block, successor *ssa.BasicBlock, reason resourceLifetimeReason) {
	if !analysis.probe.Enabled() {
		return
	}
	branch := block.Instrs[len(block.Instrs)-1]
	analysis.probe.Label(analysisTrace.Step{
		Reason: reason.String(), Outcome: analysisTrace.OutcomeUnknown,
		Pos: branch.Pos(), Function: block.Parent().String(),
		Details: map[string]string{"branch": branch.String(), "successor": strconv.Itoa(successor.Index)},
	})
}

// processExitReclaims accepts a resource that program exit genuinely cleans
// up. A leak does harm when it accumulates or when its cleanup has an effect
// that exit would lose. An acquisition that runs at most once from main.main
// cannot accumulate, and every path that leaves main ends the process, which
// closes descriptors and connections. Only contracts whose cleanup merely
// reclaims qualify: a compressor's Close flushes buffered data, a transaction
// must commit, and an inferred owner's Close is not known to be free of such
// effects, so all of those are still reported.
func processExitReclaims(call *ssa.Call, contract resourceContract, budget *proofs.SearchBudget) bool {
	switch contract.family {
	case resourceFamilyOS, resourceFamilyHTTP:
	case resourceFamilySQL:
		if slices.Contains(contract.cleanup, "Commit") {
			return false
		}
	default:
		return false
	}
	// A complete unique private caller chain keeps this process-local policy
	// out of unconditional callee facts. Repeated/escaping helpers cannot
	// establish that their acquisitions happen only once before process exit.
	// https://github.com/boxesandglue/boxesandglue/blob/79509f4b6b0e2e7a1d0562139ab4d9946d4be080/helper/main.go#L10-L35
	return ssacall.RunsOnceThroughPrivateEntryCallsWithin(call, budget)
}

// Flow setup collects bounded owner, collection and deferred-release evidence
// before the path proof starts. A possible prior release stops at uncertainty;
// partial discovery cannot supply classifiers with an authoritative census.
func (analysis *resourceAnalysis) prepareResourceFlow() resourceProof {
	return resourceProof(resourcepolicy.PrepareFlow(preparationQueries{analysis}))
}

// The resource walk keeps activation, cleanup and uncertainty on each path.
// Shared traversal and guard mechanics charge the existing candidate pool;
// incomplete evidence never retains a diagnostic witness or a release claim.

type resourceFlowState struct {
	block       *ssa.BasicBlock
	predecessor *ssa.BasicBlock
	index       int
	obligation  resourcemodel.Obligation
	// guards are the branch outcomes this path has established. An edge
	// that contradicts one is unknown, never pruned: a guard read from a
	// cell could have changed through a pointer the analysis does not see,
	// and even a stable guard's contradiction only declines to report through
	// a path the analysis cannot rule out. Mutagen guards a profiler's
	// creation and its finalization on one address-taken flag, and fortio a
	// profile file's creation and its close on one option field:
	// https://github.com/mutagen-io/mutagen/blob/6ccfeaaf4dfd261e59ef9aac56e3c157b62e605b/tools/scan_bench/main.go#L140-L172
	// https://github.com/fortio/fortio/blob/5c19725ff61c9f7ad944b91ec32d96a399341d87/fhttp/httprunner.go#L199-L215
	guards ssapath.PathGuards
}

type resourceFlowKey struct {
	location   ssapath.FlowLocationKey
	obligation resourcemodel.Obligation
}

// proveResourceFlow owns setup, coverage and cutoff availability together.
// Resource-specific activation and contradiction policy stays in this walk;
// the shared engines supply traversal mechanics, never diagnostic policy.
func (analysis *resourceAnalysis) proveResourceFlow(errorValue ssa.Value) resourceLifetimePolicyResult {
	budget := analysis.budget(resourcePoolBudget)
	index := cfg.InstructionIndexWithin(analysis.acquisition, budget)
	if resourceFlowExhausted(budget) {
		return unknownResourceLifetime(resourceReasonBudgetExhausted)
	}
	if index < 0 {
		return unknownResourceLifetime(resourceReasonAcquisitionLocationUnknown)
	}
	// Retain reachability's existing per-question cap while charging all of its
	// traversal and feasibility work to the candidate pool.
	reachBudget := analysis.budget(proofs.SummaryBudget)
	reachable := analysis.acquisitionReachable(reachBudget)
	if resourceFlowExhausted(budget) || resourceFlowExhausted(reachBudget) {
		return unknownResourceLifetime(resourceReasonBudgetExhausted)
	}
	if !reachable {
		return acceptedResourceLifetime(resourceReasonAcquisitionUnreachable)
	}
	// Begin after acquisition; predecessor and obligation state distinguish
	// error edges from paths that still owe cleanup.
	guards := ssapath.GuardsDominatingWithin(analysis.acquisition, budget)
	initial := []resourceFlowState{{block: analysis.acquisition.Block(), index: index + 1, obligation: resourcemodel.Acquired(), guards: guards}}
	opaque, leaks, incomplete := false, false, false
	cfg.WalkStatesWithin(initial, func(state resourceFlowState) resourceFlowKey { return resourceStateKey(state, budget) },
		func(state resourceFlowState) ([]resourceFlowState, bool) {
			state, leaks = advanceResourceState(analysis, state, budget)
			if resourceFlowExhausted(budget) || leaks {
				return nil, false
			}
			opaque = opaque || state.obligation.Unknown()
			edges := resourceSuccessorStates(analysis, state, errorValue, budget)
			incomplete = edges.State == proofs.EvidenceUnknown
			return edges.states, !incomplete
		}, budget)
	// Cutoff cannot retain a leak witness or an exact release claim, even when
	// a nested classifier or edge callback exhausted the candidate pool.
	if resourceFlowExhausted(budget) || incomplete {
		analysis.leak = nil
		return unknownResourceLifetime(resourceReasonBudgetExhausted)
	}
	if leaks {
		result := reportedResourceLifetime(resourceReasonUnownedReturn)
		result.leak = analysis.leak
		return result
	}
	if opaque {
		return unknownResourceLifetime(resourceReasonOpaqueConsumption)
	}
	return acceptedResourceLifetime(resourceReasonReleaseProven)
}

func resourceStateKey(state resourceFlowState, budget *proofs.SearchBudget) resourceFlowKey {
	return resourceFlowKey{
		location:   ssapath.FlowLocationKeyWithin(state.block, state.predecessor, state.index, state.guards, budget),
		obligation: state.obligation,
	}
}

// A sibling classifier query can exhaust the shared parent without spending
// through the walk's child again. Both limits must still invalidate its proof.
func resourceFlowExhausted(budget *proofs.SearchBudget) bool {
	return budget.Exhausted() || budget.PoolExhausted()
}

func advanceResourceState(analysis *resourceAnalysis, state resourceFlowState, budget *proofs.SearchBudget) (resourceFlowState, bool) {
	// A release or transfer anywhere before a return settles the path. An
	// opaque consumption does not settle it but removes the proof: the
	// return is then neither owned nor a defect.
	for _, instruction := range state.block.Instrs[state.index:] {
		if !budget.Spend() {
			return state, false
		}
		state.guards = state.guards.AfterWithin(instruction, budget)
		if resourceFlowExhausted(budget) {
			return state, false
		}
		switch analysis.action(instruction) {
		case actionSettled:
			state.obligation = state.obligation.Discharged()
		case actionUnknown:
			state.obligation = state.obligation.Uncertain()
		case actionNone:
		}
		if resourceFlowExhausted(budget) {
			return state, false
		}
		// A call that never returns, whether os.Exit or a project's own fatal
		// wrapper the summaries prove, ends this path with nothing to release.
		terminated := ssapath.InstructionTerminatesWithin(instruction, analysis.summaries.TerminatesWithin(budget), budget)
		if resourceFlowExhausted(budget) {
			return state, false
		}
		if terminated {
			state.obligation = state.obligation.Absent()
			break
		}
		returned, ok := instruction.(*ssa.Return)
		if ok && analysis.probe.Enabled() {
			analysis.probe.Evidence(analysisTrace.Step{
				Reason: resourceReasonResourceReturnPath.String(), Outcome: analysisTrace.OutcomeObserved,
				Pos: returned.Pos(), Function: returned.Parent().String(),
				Details: map[string]string{
					"active":   strconv.FormatBool(state.obligation.Active()),
					"released": strconv.FormatBool(state.obligation.Settled()),
					"unknown":  strconv.FormatBool(state.obligation.Unknown()),
				},
			})
		}
		if ok && state.obligation.Unsettled() {
			proof := analysis.proveResourceReturn(returned, budget)
			if proof.state == proofs.EvidenceUnknown {
				return state, false
			}
			if proof.state == proofs.EvidenceProven {
				analysis.leak = proof.leak
				return state, true
			}
		}
	}
	return state, false
}

type resourceSuccessorsProof struct {
	resourceProof
	states []resourceFlowState
}

func resourceSuccessorStates(
	analysis *resourceAnalysis, state resourceFlowState, errorValue ssa.Value, budget *proofs.SearchBudget,
) resourceSuccessorsProof {
	pass, resource, optionalAcquisition, candidate := analysis.pass, analysis.resource, analysis.optional, analysis.candidate
	edges := analysis.successorPolicy().EdgesWithin(state.block, state.predecessor, state.guards, budget)
	if optionalAcquisition.Proven() && state.block == optionalAcquisition.merge && state.predecessor == optionalAcquisition.acquisitionBlock {
		acquired := optionalAcquisition.acquiredSuccessor
		guards, contradiction := state.guards.ExtendWithin(state.block, acquired, nil, budget)
		edges = []ssapath.SuccessorEdge{{To: acquired, Guards: guards, Contradiction: contradiction}}
		traceOptionalAcquisition(pass, optionalAcquisition, candidate)
	}
	if resourceFlowExhausted(budget) {
		return unavailableResourceSuccessors()
	}
	result := make([]resourceFlowState, 0, len(edges))
	for _, edge := range edges {
		if !budget.Spend() {
			return unavailableResourceSuccessors()
		}
		successor := edge.To
		obligation := state.obligation
		branch := proveResourceSuccessBranch(pass, analysis.summaries, state.block, successor, errorValue, candidate,
			budget.Within(proofs.SummaryBudget))
		if branch.State == proofs.EvidenceUnknown {
			return unavailableResourceSuccessors()
		}
		if branch.Proven() {
			if !branch.success {
				obligation = obligation.Absent()
			}
		}
		presence := proveResourcePresenceBranch(state.block, state.predecessor, successor, resource, budget)
		if presence.Proven() && !presence.Present {
			obligation = obligation.Absent()
		}
		// Error and presence evidence change activation only on this edge.
		// Repeated guard contradictions instead retain an unknown obligation:
		// dropping it would turn unavailable path evidence into cleanup.
		guards, contradiction := edge.Guards, edge.Contradiction
		if contradiction != ssapath.GuardConsistent {
			obligation = obligation.Uncertain()
			analysis.traceUncertainEdge(state.block, successor, resourceReasonRepeatedGuardEdgeUnknown)
		}
		rows := proveSQLRowsExhaustionEdge(state.block, successor, resource, budget.Within(proofs.QueryBudget))
		if rows.State == proofs.EvidenceUnknown {
			return unavailableResourceSuccessors()
		}
		if rows.Proven() {
			obligation = obligation.Uncertain()
			analysis.traceUncertainEdge(state.block, successor, resourceReasonRowsExhaustedEdgeUnknown)
		}
		if analysis.collection.releasedOnEdge(state.block, successor) {
			obligation = obligation.Discharged()
			analysis.traceCollectionReleased(state.block, successor)
		}
		// A conditional helper settles only the edge selected by its result.
		// Optional-acquisition phis retain their own stricter cleanup policy.
		if !obligation.Settled() && !optionalAcquisition.Proven() {
			if analysis.evidence.CompletionOnEdge(state.block, successor, lifecycle.CompletionRequest{
				Target: resource, Methods: analysis.contract.cleanup, Budget: analysis.budget(1000),
			}).Proven() {
				obligation = obligation.Discharged()
			}
		}
		if resourceFlowExhausted(budget) {
			return unavailableResourceSuccessors()
		}
		result = append(result, resourceFlowState{
			block: successor, predecessor: state.block, obligation: obligation, guards: guards,
		})
	}
	return resourceSuccessorsProof{resourceProof: resourceProof{State: proofs.EvidenceProven}, states: result}
}

func unavailableResourceSuccessors() resourceSuccessorsProof {
	return resourceSuccessorsProof{resourceProof: resourceProof{State: proofs.EvidenceUnknown, Reason: resourceReasonBudgetExhausted}}
}

// Evaluation contexts carry the reached phase's inputs explicitly. Begin is the
// only constructor of candidate tracing and allowances; initialization retains
// the correlation packet without exposing its search representation to policy.
type evaluationQueries struct {
	pass     *analysis.Pass
	evidence *lifecyclefacts.LifecycleEvidence
	call     *ssa.Call
	resource ssa.Value
	contract resourceContract
}

func (q evaluationQueries) MemoryWriter() bool { return memoryWriterExempt(q.call, q.contract) }

func (q evaluationQueries) Begin() resourcepolicy.AcquisitionQueries {
	q.evidence.ForCandidate(q.call.Pos())
	probe := analysisTrace.For(q.pass, "resourcelifetime", string(check.ResourceRelease), q.call.Pos())
	pool := proofs.NewSearchBudget(resourcePoolBudget).Observed(probe.Observer())
	return acquisitionQueries{evaluationQueries: q, probe: probe, pool: pool}
}

type acquisitionQueries struct {
	evaluationQueries
	probe analysisTrace.Probe
	pool  *proofs.SearchBudget
}

func (q acquisitionQueries) Error() resourcepolicy.PairedError {
	result := proveAcquisitionErrorResultWithin(q.call, q.pool.Within(releaseSearchBudget))
	return resourcepolicy.PairedError{State: result.proof.State, Reason: result.proof.Reason, Value: result.value}
}

func (q acquisitionQueries) Boundary() resourcepolicy.Reason {
	return httpAcquisitionBoundary(q.pass, q.call, q.pool.Within(releaseSearchBudget))
}

func (q acquisitionQueries) Canceled() resourcepolicy.Evidence {
	return resourcepolicy.Evidence(proveAcquisitionContextCanceledWithin(q.call, q.pool.Within(releaseSearchBudget)))
}

func (q acquisitionQueries) Asserted(errorValue ssa.Value) resourcepolicy.Evidence {
	proof := proveAcquisitionErrorWithin(q.call, q.resource, errorValue, q.contract.packagePath == "net/http", q.pool.Within(releaseSearchBudget))
	return resourcepolicy.Evidence(proof)
}

func (q acquisitionQueries) Optional(errorValue ssa.Value) resourcepolicy.Correlation {
	optional := proveOptionalAcquisitionWithin(q.call, q.resource, errorValue, q.pool.Within(releaseSearchBudget))
	return resourcepolicy.Correlation{
		State: optional.proof.State, Reason: optional.proof.Reason,
		Initialize: initializationQueries{acquisitionQueries: q, optional: optional},
	}
}

type initializationQueries struct {
	acquisitionQueries
	optional optionalAcquisitionProof
}

func (q initializationQueries) Prepare() resourcepolicy.Prepared {
	resource := q.resource
	if q.optional.Proven() {
		resource = q.optional.resourcePhi
	}
	a := &resourceAnalysis{
		acquisition: q.call, summaries: resourceSummaries.Provider(q.pass), pass: q.pass, evidence: q.evidence,
		function: q.call.Parent(), resource: resource, candidate: q.call.Pos(), contract: q.contract, optional: q.optional,
		actions: map[ssa.Instruction]resourceAction{}, probe: q.probe, pool: q.pool,
	}
	setup := a.prepareResourceFlow()
	return resourcepolicy.Prepared{State: setup.State, Reason: setup.Reason, Queries: evaluationFlow{analysis: a}}
}

type evaluationFlow struct{ analysis *resourceAnalysis }

func (q evaluationFlow) Flow(errorValue ssa.Value) resourcepolicy.Decision {
	result := q.analysis.proveResourceFlow(errorValue)
	return resourcepolicy.Decision{State: result.state, Reason: result.reason, Leak: result.leak}
}

func (q evaluationFlow) DatabaseCleanup() bool {
	a := q.analysis
	return sqlDatabaseCall(a.acquisition.Common(), "Prepare", "PrepareContext") && lifecycle.ProveEnclosingCompletion(lifecycle.EnclosingCompletionRequest{
		Function: a.acquisition.Parent(), Value: ssaflow.CallReceiver(a.acquisition.Common()), Methods: []string{"Close"}, Budget: a.budget(10000),
	}).Proven()
}
func (q evaluationFlow) PoolExhausted() bool { return q.analysis.pool.Exhausted() }
func (q evaluationFlow) Exit() resourcepolicy.ExitEvidence {
	budget := q.analysis.budget(releaseSearchBudget)
	reclaimed := processExitReclaims(q.analysis.acquisition, q.analysis.contract, budget)
	return resourcepolicy.ExitEvidence{Reclaimed: reclaimed, Exhausted: budget.Exhausted()}
}

// Preparation queries own the classifier input mutations and immediate evidence
// observations. The policy selects their order and conservative stop points.
type preparationQueries struct{ analysis *resourceAnalysis }

func (q preparationQueries) Deferred() resourcepolicy.Evidence {
	a := q.analysis
	return resourcepolicy.Evidence(a.proveDeferredBeforeAcquisitionWithin(a.acquisition, a.budget(releaseSearchBudget)))
}

func (q preparationQueries) Owners() bool {
	return q.analysis.discoverResourceOwnersWithin(q.analysis.budget(releaseSearchBudget)).Proven()
}

func (q preparationQueries) Guarded() resourcepolicy.Evidence {
	a := q.analysis
	a.collection = a.localCollection()
	return resourcepolicy.Evidence(a.discoverResultGuardedDefersWithin(a.budget(releaseSearchBudget)))
}

func (q preparationQueries) Prior() resourcepolicy.Evidence {
	a := q.analysis
	prior := a.provePriorCleanupWithin(a.acquisition, a.budget(releaseSearchBudget))
	if prior.Proven() {
		a.emitAction(prior.Instruction, actionUnknown, prior.Reason)
	}
	return resourcepolicy.Evidence(prior.resourceProof)
}
