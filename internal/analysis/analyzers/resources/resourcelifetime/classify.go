package resourcelifetime

import (
	"go/token"
	"go/types"

	"github.com/kojah/gohawk/internal/analysis/analyzers/resources/resourcelifetime/resourcepolicy"
	"github.com/kojah/gohawk/internal/analysis/analyzers/resources/resourcelifetime/sqlparents"
	"github.com/kojah/gohawk/internal/analysis/passes/lifecyclefacts"
	"github.com/kojah/gohawk/internal/analysis/summaries"
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/lifecycle"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/syntax"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	analysisTrace "github.com/kojah/gohawk/internal/reporting/trace"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ssa"
)

// The classifier labels each instruction after an acquisition once, and the
// flow asks one question of the labels: does a release or transfer cover
// every feasible return, and if not, did anything the analysis cannot see
// through consume the resource on that path? A diagnostic needs the second
// answer to be no. Summaries are the model's knowledge: a summarized callee
// that neither releases, stores, nor owns the resource is transparent, so a
// read helper leaves the obligation in place. Unknown is reserved for real
// boundaries: an interface method or function value that receives the
// resource here, a callee with neither body nor summary, a literal capturing
// the resource that is launched or deferred without a proven release, and a
// send or append that hands it to storage the analysis does not track.

type resourceAction uint8

const (
	actionNone resourceAction = iota
	// actionSettled covers both a proven release and a proven transfer; the
	// flow treats them alike and the evidence trace records which one held.
	actionSettled
	actionUnknown
)

// resourceAnalysis holds one acquisition's inputs and its memoized labels.
type resourceAnalysis struct {
	acquisition *ssa.Call
	pass        *analysis.Pass
	evidence    *lifecyclefacts.LifecycleEvidence
	summaries   *summaries.Provider
	function    *ssa.Function
	resource    ssa.Value
	// candidate identifies the acquisition every step of this proof serves, and
	// probe tags every trace event with it so one acquisition's proof can be
	// read without the interleaved steps of the others in the same function.
	candidate token.Pos
	probe     analysisTrace.Probe
	owners    []ssa.Value
	// collection is the local slice the resource was appended to, when every
	// use of it is understood; see local_collections.go.
	collection *localCollection
	// guardedDefers are the deferred literals whose release turns on a named
	// result; see result_guarded_defers.go.
	guardedDefers []lifecycle.ResultGuard
	contract      resourceContract
	optional      optionalAcquisitionProof
	actions       map[ssa.Instruction]resourceAction
	// pool is this acquisition's total across every query its proof asks;
	// see budget.
	pool *proofs.SearchBudget
	// stores reuses one destination proof between release and opacity labels.
	stores map[*ssa.Store]resourceStorageProof
	// wrappers shares return classification evidence with owner disposition.
	wrappers map[*ssa.Return]resourceReturnedWrapperProof
	// leak is the return at which the flow walk found the resource owed.
	leak *ssa.Return
}

// resourcePoolBudget bounds the pooled queries of one acquisition proof. The largest single
// question, a release search, may itself cost releaseSearchBudget, so the
// pool allows four of them; a proof that needs more is a pathological
// candidate, and an exhausted pool is unknown exactly as an exhausted
// search is.
const resourcePoolBudget = 4 * releaseSearchBudget

// budget draws one shared query's allowance from this candidate's pool:
// give-ups inside it reach the probe, so a trace of the acquisition shows
// where shared storage, summary, or completion evidence ran out, and these
// queries stay bounded together.
func (analysis *resourceAnalysis) budget(limit int) *proofs.SearchBudget {
	if analysis.pool == nil {
		analysis.pool = proofs.NewSearchBudget(resourcePoolBudget).Observed(analysis.probe.Observer())
	}
	return analysis.pool.Within(limit)
}

func (analysis *resourceAnalysis) action(instruction ssa.Instruction) resourceAction {
	if action, ok := analysis.actions[instruction]; ok {
		return action
	}
	action, reason := analysis.classify(instruction)
	analysis.actions[instruction] = action
	analysis.emitAction(instruction, action, reason)
	return action
}

// classify labels the instruction and names why. The reason is the label for
// a settled or untouched resource, and for an opaque one it is the boundary
// that stopped the proof, so a reader can tell an interface call from a
// callee with no body without rereading this code.
func (analysis *resourceAnalysis) classify(instruction ssa.Instruction) (resourceAction, resourceLifetimeReason) {
	return classifyResource(analysis, instruction)
}

// Classifier queries keep the existing allowance owner, query limit and exact
// instruction. They run only when the ordered rules reach that evidence family.
type resourceClassifySignature func(*resourceAnalysis, ssa.Instruction) (resourceAction, resourceLifetimeReason)

var _ resourceClassifySignature = classifyResource

const (
	resourceProven    = proofs.EvidenceProven
	resourceDisproven = proofs.EvidenceDisproven
	resourceUnknown   = proofs.EvidenceUnknown
)

func (analysis *resourceAnalysis) parentCleanup(instruction ssa.Instruction) resourceProof {
	return proveSQLParentCleanupWithin(analysis.acquisition, instruction, analysis.budget(proofs.SummaryBudget))
}

func (analysis *resourceAnalysis) ambiguousCleanup(instruction ssa.Instruction) resourceProof {
	return analysis.proveAmbiguousCleanupWithin(instruction, ssaflow.InstructionCall(instruction), analysis.budget(releaseSearchBudget))
}

func (analysis *resourceAnalysis) pairedCleanup(instruction ssa.Instruction) resourceProof {
	return analysis.provePairedErrorCleanupWithin(instruction, ssaflow.InstructionCall(instruction), analysis.budget(releaseSearchBudget))
}

func (analysis *resourceAnalysis) loopCleanup(instruction ssa.Instruction) resourceProof {
	return analysis.proveImportedLoopReleaseWithin(instruction, ssaflow.InstructionCall(instruction), analysis.budget(proofs.SummaryBudget))
}

// releaseLabel preserves the distinction between cleanup and an abandoned
// search. Both suppress a leak, but only positive completion evidence settles
// the obligation. A loop-only witness is likewise uncertain; conditional
// helpers with complete path evidence retain the ordinary classification.
func releaseLabel(proof lifecyclefacts.Proof) (resourceAction, resourceLifetimeReason) {
	if proof.Proven() {
		return actionSettled, resourceReasonSettled
	}
	switch proof.Reason {
	case proofs.EvidenceBudgetExhausted:
		return actionUnknown, resourceReasonBudgetExhausted
	case proofs.EvidenceCompletionInCycle:
		return actionUnknown, resourceReasonHelperCleanupInLoop
	default:
		return actionNone, resourceReasonNone
	}
}

// Compressors own finalization, not their output descriptor. An error return
// may abandon that output rather than publish it; this local proof cannot
// decide the caller's policy. Successful returns still owe finalization.
// https://github.com/goreleaser/nfpm/blob/3627b6a6466c0ae3bbe17fe7b98710c36e750907/rpm/srpm.go#L123
func (analysis *resourceAnalysis) compressionOutputAbandoned(instruction ssa.Instruction) bool {
	if analysis.contract.family != resourceFamilyCompression {
		return false
	}
	if returned, ok := instruction.(*ssa.Return); ok {
		for _, result := range returned.Results {
			if types.Identical(result.Type(), types.Universe.Lookup("error").Type()) && !ssaflow.DefinitelyNil(result) {
				return true
			}
		}
	}
	common := ssaflow.InstructionCall(instruction)
	return ssacall.CallMatchesSymbol(common, syntax.PackageMethod(syntax.MethodSymbol{
		PackagePath: "io", Receiver: "PipeWriter", Name: "CloseWithError",
	})) && len(common.Args) == 2 && !ssaflow.DefinitelyNil(common.Args[1]) &&
		// Possible identity is sufficient for uncertainty, including repeated
		// loads of a captured pipe across a wait. This never proves release.
		heapmodel.MayAlias(ssaflow.CallReceiver(common), analysis.acquisition.Common().Args[0])
}

// opaqueConsumption reports whether the instruction hands the resource to
// something the analysis cannot see through.
func (analysis *resourceAnalysis) opaqueConsumption(instruction ssa.Instruction) (resourceLifetimeReason, bool) {
	return resourcepolicy.Consume(opacityQueries{analysis}, instruction)
}

func (analysis *resourceAnalysis) opaqueCall(instruction ssa.Instruction, common *ssa.CallCommon) (resourceLifetimeReason, bool) {
	return resourcepolicy.Call(opacityQueries{analysis}, instruction, common)
}

// Named and dynamically selected functions share the argument-level boundary;
// literal captures and interface receivers are classified by opaqueCall first.
func (analysis *resourceAnalysis) opaqueFunctionCall(instruction ssa.Instruction, common *ssa.CallCommon, carried bool) (resourceLifetimeReason, bool) {
	return resourcepolicy.Function(opacityQueries{analysis}, instruction, common, carried)
}

// SQL parent cleanup supplies uncertainty rather than synchronous release. Both
// classification and prior-defer evidence use the same exact relationship proof.
func proveSQLParentCleanupWithin(acquisition *ssa.Call, instruction ssa.Instruction, budget *proofs.SearchBudget) resourceProof {
	parent := sqlparents.ProveWithin(acquisition, instruction, budget)
	if parent.State == proofs.EvidenceUnknown {
		return resourceProof{State: parent.State, Reason: resourceReasonBudgetExhausted}
	}
	reason := resourceReasonUntouched
	switch parent.Kind {
	case sqlparents.Unrelated:
		reason = resourceReasonUntouched
	case sqlparents.DatabaseClosed:
		reason = resourceReasonStatementParentClosed
	case sqlparents.TransactionFinished:
		reason = resourceReasonRowsTransactionFinished
	}
	return resourceProof{State: parent.State, Reason: reason}
}

// opacityQueries supplies reached evidence without changing its allowance,
// observation point or cache ownership. The rules own the decision ordering.
type opacityQueries struct{ analysis *resourceAnalysis }

func (q opacityQueries) Returned(value *ssa.Return) resourcepolicy.Evidence {
	return resourcepolicy.Evidence(q.analysis.returnedWrapperWithin(value, q.analysis.budget(proofs.SummaryBudget)).resourceProof)
}

func (q opacityQueries) Storage(value *ssa.Store) resourcepolicy.Storage {
	p := q.analysis.resourceStorage(value)
	return resourcepolicy.Storage{State: p.State, Reason: p.Reason, Owner: p.Owner}
}

func (q opacityQueries) Wrapper(value *ssa.Store) resourcepolicy.Evidence {
	return resourcepolicy.Evidence(q.analysis.proveWrapperStoredOnForeignOwnerWithin(value, q.analysis.budget(proofs.SummaryBudget)))
}

func (q opacityQueries) Handoff(value ssa.Value, instruction ssa.Instruction) resourcepolicy.Evidence {
	return resourcepolicy.Evidence(q.analysis.responseBodyAggregateHandoff(value, instruction))
}

func (q opacityQueries) Payload(value ssa.Value, reason resourcepolicy.Reason) (resourcepolicy.Reason, bool) {
	return q.analysis.carriedPayload(value, reason)
}

func (q opacityQueries) Select(value *ssa.Select) (resourcepolicy.Reason, bool) {
	for _, state := range value.States {
		if state.Send != nil {
			if reason, opaque := resourcepolicy.Send(q, state.Send, value); opaque {
				return reason, true
			}
		}
	}
	return resourceReasonNone, false
}

func (q opacityQueries) Call(instruction ssa.Instruction, common *ssa.CallCommon) (resourcepolicy.Reason, bool) {
	return q.analysis.opaqueCall(instruction, common)
}

func (q opacityQueries) Carried(common *ssa.CallCommon) resourcepolicy.Evidence {
	return resourcepolicy.Evidence(q.analysis.proveCarriedArgumentsWithin(common, q.analysis.budget(proofs.SummaryBudget)))
}

func (q opacityQueries) Callback(instruction ssa.Instruction, common *ssa.CallCommon) resourcepolicy.Evidence {
	return resourcepolicy.Evidence(q.analysis.provePossiblyRetainedCallbackWithin(instruction, common, q.analysis.budget(proofs.SummaryBudget)))
}

func (q opacityQueries) Exposure(instruction ssa.Instruction, common *ssa.CallCommon) resourcepolicy.Evidence {
	return resourcepolicy.Evidence(q.analysis.proveAsynchronousExposureWithin(instruction, common, q.analysis.budget(proofs.SummaryBudget)))
}

func (q opacityQueries) Closure(instruction ssa.Instruction, closure *ssa.MakeClosure, carried bool) (resourcepolicy.Reason, bool) {
	return q.analysis.opaqueClosureCall(instruction, closure, carried)
}

func (q opacityQueries) Function(instruction ssa.Instruction, common *ssa.CallCommon, carried bool) (resourcepolicy.Reason, bool) {
	return q.analysis.opaqueFunctionCall(instruction, common, carried)
}

func (q opacityQueries) Chain(instruction ssa.Instruction, common *ssa.CallCommon) resourcepolicy.Evidence {
	return resourcepolicy.Evidence(q.analysis.proveChainKeptWithin(instruction, common, q.analysis.budget(proofs.SummaryBudget)))
}

func (q opacityQueries) Escape(instruction ssa.Instruction, common *ssa.CallCommon) resourcepolicy.Evidence {
	return resourcepolicy.Evidence(q.analysis.proveAggregateOwnerEscapeWithin(instruction, common, q.analysis.budget(proofs.SummaryBudget)))
}

func (q opacityQueries) Aggregate(common *ssa.CallCommon) resourcepolicy.Evidence {
	return resourcepolicy.Evidence(q.analysis.proveCarriedAggregateArgumentsWithin(common, q.analysis.budget(proofs.SummaryBudget)))
}

func (q opacityQueries) Publication(instruction ssa.Instruction) resourcepolicy.Evidence {
	return resourcepolicy.Evidence(proveCallResultMayTransferWithin(instruction, q.analysis.budget(proofs.SummaryBudget)))
}

// Unmodeled asks only about unavailable body/summary evidence. A readable
// body was already judged by completion, and a summarized no-effect callee is
// transparent. Neither case becomes opaque merely for lacking a release here.
func (q opacityQueries) Unmodeled(instruction ssa.Instruction, common *ssa.CallCommon) bool {
	return !q.analysis.evidence.CalleeSummarized(instruction) && len(common.StaticCallee().Blocks) == 0
}
