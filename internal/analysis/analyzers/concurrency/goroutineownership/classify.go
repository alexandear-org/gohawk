package goroutineownership

import (
	"go/types"
	"slices"

	"github.com/kojah/gohawk/internal/analysis/analyzers/concurrency/goroutineownership/containment"

	"github.com/kojah/gohawk/internal/analysis/analyzers/concurrency/goroutineownership/receives"
	proofs "github.com/kojah/gohawk/internal/engine/proof"

	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	ssapath "github.com/kojah/gohawk/internal/engine/ssaflow/path"
	analysisTrace "github.com/kojah/gohawk/internal/reporting/trace"
	"golang.org/x/tools/go/ssa"
)

// The classifier labels each instruction of the spawning function with the
// effect it has on the worker's tracked values. A join is an exact observation
// of completion; a transfer hands the exact value to a caller or an owner that
// outlives the function; unknown covers any other consumption the analysis
// cannot see through, such as an opaque call, a send, or a helper that lets
// the value escape. Anything that does not touch a tracked value is none.
// Unknown is deliberately not a weaker join: it suppresses the diagnostic
// instead of proving ownership.

type ownershipAction uint8

const (
	actionNone ownershipAction = iota
	actionJoin
	actionTransfer
	actionUnknown
)

func (action ownershipAction) String() string {
	switch action {
	case actionJoin:
		return "join"
	case actionTransfer:
		return "transfer"
	case actionUnknown:
		return "opaque-use"
	case actionNone:
	}
	return "none"
}

// obligation maps this classifier's labels onto the shared flow lattice: a
// join or transfer is exact evidence and an opaque use is unknown.
func (action ownershipAction) obligation() ssapath.ObligationAction {
	switch action {
	case actionJoin, actionTransfer:
		return ssapath.ObligationExact
	case actionUnknown:
		return ssapath.ObligationUnknown
	case actionNone:
	}
	return ssapath.ObligationNone
}

func (analysis *spawnAnalysis) obligation(instruction ssa.Instruction) ssapath.ObligationAction {
	return analysis.action(instruction).obligation()
}

// edgeObligation credits a selected receive of a tracked signal as an exact
// join on that arm alone, and a selected receive of an opaque worker's
// context as an opaque observation on that arm alone. The exit of a counted
// select drain is an exact join on that edge alone.
func (analysis *spawnAnalysis) edgeObligation(from, to *ssa.BasicBlock) ssapath.ObligationAction {
	if analysis.selectedJoinEdge(from, to) {
		return ssapath.ObligationExact
	}
	if drain := analysis.countedDrainAction(from, to); drain != ssapath.ObligationNone {
		return drain
	}
	if channel, selected := ssapath.SelectedReceiveOnEdge(from, to); selected && analysis.possibleSignal(channel) {
		analysis.recordEdge(from, to, reasonSelectedPossibleReceiveEdge, ssapath.ObligationUnknown)
		return ssapath.ObligationUnknown
	}
	if analysis.selectedOwnershipEdge(from, to) {
		return ssapath.ObligationUnknown
	}
	return ssapath.ObligationNone
}

// strongerAction merges the labels of several tracked values touched by one
// instruction. A proven join wins because it already required every-return
// coverage; otherwise any escape keeps the instruction opaque.
func strongerAction(current, next ownershipAction) ownershipAction {
	switch {
	case current == actionJoin || next == actionJoin:
		return actionJoin
	case current == actionUnknown || next == actionUnknown:
		return actionUnknown
	case current == actionTransfer || next == actionTransfer:
		return actionTransfer
	default:
		return actionNone
	}
}

func (analysis *spawnAnalysis) action(instruction ssa.Instruction) ownershipAction {
	if action, ok := analysis.actions[instruction]; ok {
		return action
	}
	action, reason := analysis.classify(instruction)
	analysis.actions[instruction] = action
	analysis.traceLabel(instruction, action, reason)
	return action
}

// traceLabel records a join, transfer, or opaque-use label once, when the
// instruction is first classified, so the trace lists labels in the order the
// walk met them, each with the rule that produced it. An instruction labelled
// none is not traced.
func (analysis *spawnAnalysis) traceLabel(instruction ssa.Instruction, action ownershipAction, reason goroutineOwnershipReason) {
	if action == actionNone || !analysis.probe.Enabled() {
		return
	}
	outcome := analysisTrace.OutcomeAccepted
	if action == actionUnknown {
		outcome = analysisTrace.OutcomeUnknown
	}
	analysis.probe.Label(analysisTrace.Step{
		Reason: reason.String(), Outcome: outcome, Pos: instruction.Pos(), Function: analysis.function.String(),
		Details: map[string]string{"instruction": instruction.String(), "label": action.String()},
	})
}

// classify labels the instruction and names the rule that decided it, so a
// trace of an opaque use says which boundary it was.
func (analysis *spawnAnalysis) classify(instruction ssa.Instruction) (ownershipAction, goroutineOwnershipReason) {
	action, reason, recognized := recognizeSpawnEntry(analysis, instruction)
	if recognized {
		return action, reason
	}
	// These terminal proof queries dispatch by SSA type. Their implementation
	// retains return precedence, external-store identity and call contracts.
	switch typed := instruction.(type) {
	case *ssa.Return:
		// A merged return or guarded re-walk uses the same instruction cache;
		// neither the query nor its label should be repeated.
		return analysis.returnAction(typed)
	case *ssa.Store:
		return analysis.storeAction(typed), reasonLabelStoredOutside
	case *ssa.Call, *ssa.Defer, *ssa.Go:
		return analysis.callAction(instruction, ssaflow.InstructionCall(instruction))
	}
	return classifySpawnInstruction(analysis, instruction)
}

type (
	classifySignature  func(analysis *spawnAnalysis, instruction ssa.Instruction) (ownershipAction, goroutineOwnershipReason)
	recognizeSignature func(analysis *spawnAnalysis, instruction ssa.Instruction) (ownershipAction, goroutineOwnershipReason, bool)
)

var (
	_ classifySignature  = classifySpawnInstruction
	_ recognizeSignature = recognizeSpawnEntry
)

func storedTerminationInstruction(instruction ssa.Instruction) bool {
	call, ok := instruction.(*ssa.Call)
	return ok && storedTerminationReceiver(call.Common())
}

func (analysis *spawnAnalysis) selectedSignalEntry(instruction ssa.Instruction) bool {
	return receives.SelectedAtEntry(instruction, analysis.isSignal)
}

func (analysis *spawnAnalysis) selectedPossibleEntry(instruction ssa.Instruction) bool {
	return receives.SelectedAtEntry(instruction, analysis.possibleSignal)
}

func (analysis *spawnAnalysis) guaranteedSignal(instruction ssa.Instruction) bool {
	return receives.Guaranteed(instruction, analysis.isSignal)
}

func (analysis *spawnAnalysis) guaranteedPossible(instruction ssa.Instruction) bool {
	return receives.Guaranteed(instruction, analysis.possibleSignal)
}

func (analysis *spawnAnalysis) aggregateReceive(instruction ssa.Instruction) bool {
	return receives.From(instruction, analysis.signalAggregateCarries)
}

// Capturing a strict testing object introduces a receiver load that the
// storage-free contract registry deliberately cannot resolve. Ask the existing
// storage proof for its immutable origin, then delegate policy to that registry.
// This is a terminal-return boundary, never an assertion that the worker joined.
// https://github.com/ConduitIO/conduit/blob/9946a19b9fff997675f78bbc5ff437e760d39f4f/pkg/lifecycle/stream/destination_acker_test.go#L30-L92
func storedTerminationReceiver(common *ssa.CallCommon) bool {
	receiver := ssaflow.CallReceiver(common)
	if receiver == nil || len(common.Args) == 0 || common.Args[0] != receiver {
		return false
	}
	resolved := heapmodel.NewStorage(nil).Resolve(receiver)
	if !resolved.Proven() || resolved.Value == receiver {
		return false
	}
	copy := *common
	copy.Args = slices.Clone(common.Args)
	copy.Args[0] = resolved.Value
	return ssacall.HasLibraryContract(&copy, ssacall.ContractTestingTermination)
}

// An exact returned handle covers the obligation even if another result has
// opaque containment. A possible handle alone never establishes ownership.
func (analysis *spawnAnalysis) returnAction(returned *ssa.Return) (ownershipAction, goroutineOwnershipReason) {
	result := actionNone
	for _, value := range returned.Results {
		action := analysis.transferAction(value)
		if action == actionTransfer {
			return actionTransfer, reasonLabelReturnedTracked
		}
		result = strongerAction(result, action)
	}
	if result == actionUnknown {
		return actionUnknown, reasonLabelReturnedContainment
	}
	if analysis.returnMayTransfer(returned) {
		return actionUnknown, reasonLabelReturnedProjection
	}
	return actionNone, reasonNone
}

// A signal mapped only to its captured aggregate may leave through a field
// projection. Matching the root establishes possible handoff, not the exact
// field or a guaranteed join, so only the unknown-return query uses it.
// https://github.com/mysteriumnetwork/node/blob/c45527af1ea80300ae3d9c92bd37255335b6140d/session/pingpong/hermes_promise_handler.go#L120-L140
func (analysis *spawnAnalysis) returnMayTransfer(returned *ssa.Return) bool {
	return slices.ContainsFunc(returned.Results, func(result ssa.Value) bool {
		root := aggregateRoot(result)
		return slices.ContainsFunc(analysis.signals, func(signal ssa.Value) bool {
			return !ssaflow.ChannelType(signal) && heapmodel.MayAlias(root, aggregateRoot(signal))
		})
	})
}

// A store outside the function transfers only an exact tracked value. Possible
// containment is opaque ownership; local storage changes nothing until handoff.
func (analysis *spawnAnalysis) storeAction(store *ssa.Store) ownershipAction {
	switch address := store.Addr.(type) {
	case *ssa.Global, *ssa.FreeVar:
		return analysis.transferAction(store.Val)
	case *ssa.FieldAddr:
		if ssaflow.ExternallyOwnedValue(address.X) {
			return analysis.transferAction(store.Val)
		}
	}
	return actionNone
}

func (analysis *spawnAnalysis) selectedJoinEdge(from, to *ssa.BasicBlock) bool {
	channel, selected := ssapath.SelectedReceiveOnEdge(from, to)
	joined := selected && analysis.isSignal(channel)
	if joined {
		analysis.recordEdge(from, to, reasonSelectedReceiveEdge, ssapath.ObligationExact)
	}
	return joined
}

// recordEdge keeps the authoritative reason and action for the trace. Tracing
// projects the action instead of reconstructing proof strength from reason names.
func (analysis *spawnAnalysis) recordEdge(from, to *ssa.BasicBlock, reason goroutineOwnershipReason, action ssapath.ObligationAction) {
	if !analysis.tracing {
		return
	}
	if analysis.edgeEvidence == nil {
		analysis.edgeEvidence = make(map[[2]int]joinEdgeEvidence)
	}
	analysis.edgeEvidence[[2]int{from.Index, to.Index}] = joinEdgeEvidence{reason: reason, action: action}
}

// selectSends reports whether a select statement offers a tracked value on a
// send case; like a plain send, the receiver may retain it.
func (analysis *spawnAnalysis) selectSends(instruction ssa.Instruction) bool {
	choice, ok := instruction.(*ssa.Select)
	if !ok {
		return false
	}
	return slices.ContainsFunc(choice.States, func(state *ssa.SelectState) bool {
		return state.Dir == types.SendOnly && analysis.consumes(state.Send)
	})
}

// isSignal requires exact channel identity. An aggregate root or a phi with
// one matching alternative explains possible completion, never an exact join.
func (analysis *spawnAnalysis) isSignal(value ssa.Value) bool {
	return analysis.isSignalWithin(value, analysis.budget())
}

// isSignalWithin shares exact channel identity between observations and relay
// discovery. Aggregate roots retain possible ownership, never an exact join.
func (analysis *spawnAnalysis) isSignalWithin(value ssa.Value, budget *proofs.SearchBudget) bool {
	if !ssaflow.ChannelType(value) {
		return false
	}
	storage := heapmodel.NewStorage(budget)
	for _, signal := range analysis.signals {
		if !budget.Spend() {
			return false
		}
		if ssaflow.ChannelType(signal) && storage.Same(value, signal).Proven() {
			return true
		}
	}
	return false
}

// possibleSignal retains the previous broad receive boundary for unknown
// ownership and guarded joins. Matching sibling fields/indexes does not show
// that the caller observed this worker's completion handle.
func (analysis *spawnAnalysis) possibleSignal(value ssa.Value) bool {
	if heapmodel.MayAliasAny(value, analysis.signals) {
		return true
	}
	root := aggregateRoot(value)
	return root != value && slices.ContainsFunc(analysis.signals, func(signal ssa.Value) bool {
		if !ssaflow.ChannelType(signal) {
			// Captured slice cells and their loaded slice share an aggregate root.
			// Index correlation stays unknown under countedJoin, not proven exact.
			// https://github.com/bazel-contrib/buildtools/blob/933e9bbe17f7619afaca1dd58ce22810042f1c13/buildifier/buildifier.go#L241-L268
			return heapmodel.MayAlias(ssaflow.CapturedBindingValue(root), ssaflow.CapturedBindingValue(aggregateRoot(signal)))
		}
		signalRoot := aggregateRoot(signal)
		return signalRoot != signal && heapmodel.MayAlias(root, signalRoot)
	})
}

// A worker-side field can resolve only to the constructor's aggregate result,
// while its caller receives the original channel passed to that constructor.
// Containment cannot identify the exact field, so this is unknown, not a join.
// https://github.com/lotusirous/go-concurrency-patterns/blob/791337ff11e69cd9d1a8587e53474a8469b3510f/17-ring-buffer-channel/main.go#L49-L62
func (analysis *spawnAnalysis) signalAggregateCarries(value ssa.Value) bool {
	return slices.ContainsFunc(analysis.signals, func(signal ssa.Value) bool {
		return !ssaflow.ChannelType(signal) && containment.Matches(ssaflow.NewReachingWalk(carryForms), signal, value)
	})
}

// carryForms are the wrappers transparent to this classifier's containment evidence.
const carryForms = ssaflow.TransparentChangeInterface | ssaflow.TransparentChangeType | ssaflow.TransparentConvert | ssaflow.TransparentMakeInterface

// transferAction reserves exact ownership for one tracked value on every
// reaching path. Containment includes old stores, mixed phis and wrapping call
// arguments, so it establishes only a possible handoff. Budget exhaustion must
// also stay unknown rather than turn an unfinished identity query into absence.
func (analysis *spawnAnalysis) transferAction(value ssa.Value) ownershipAction {
	storage := heapmodel.NewStorage(analysis.budget())
	for _, tracked := range analysis.tracked {
		if storage.Same(value, tracked.value).Proven() {
			return actionTransfer
		}
	}
	if storage.Budget().Exhausted() || analysis.consumes(value) {
		return actionUnknown
	}
	return actionNone
}

// consumes reports whether value is a tracked value or something that carries
// one: a closure capturing it, an aggregate holding it, a loaded composite, or
// the result of a call that received it.
func (analysis *spawnAnalysis) consumes(value ssa.Value) bool {
	return slices.ContainsFunc(analysis.tracked, func(tracked trackedValue) bool {
		target := tracked.value
		// A worker may receive a loaded field while its caller returns the
		// aggregate initialized with that field. Resolve the read at its own
		// instruction, rather than expecting the return to contain the load.
		if resolved := heapmodel.NewStorage(analysis.budget()).Resolve(target); resolved.Proven() {
			target = resolved.Value
		}
		return containment.Matches(ssaflow.NewReachingWalk(carryForms), value, target)
	})
}
