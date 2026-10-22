package lifecyclefacts

import (
	"go/token"
	"strconv"
	"strings"

	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/lifecycle"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	analysisTrace "github.com/kojah/gohawk/internal/reporting/trace"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ssa"
)

// Proof retains the underlying evidence and the summary rule, when applicable.
// SummaryReason never replaces an SSA reason with a string from another domain.
type Proof struct {
	proofs.Proof
	SummaryReason Reason
}

// CompletionProof retains path coverage along with its summary explanation.
type CompletionProof struct {
	proofs.CompletionProof
	SummaryReason Reason
}

// LifecycleEvidence combines memoized local SSA evidence with lifecycle summaries
// imported through the prerequisite analyzer. One evidence context belongs to one source
// function and is not safe for concurrent use.
type LifecycleEvidence struct {
	pass     *analysis.Pass
	analyzer string
	check    string
	// probe attributes each traced proof step to the candidate being judged.
	// It starts unattributed so an analyzer that never scopes still traces.
	probe analysisTrace.Probe
	local lifecycle.LocalEvidence
	// retentions answers retention questions about function literals, which
	// carry no summary of their own. It is built on first use because most
	// analyzers never ask.
	retentions *retentionCache
}

// NewLifecycleEvidence constructs evidence whose accepted, rejected, and unknown
// results use the supplied analyzer identity for structured tracing.
func NewLifecycleEvidence(pass *analysis.Pass, analyzer, check string) *LifecycleEvidence {
	evidence := &LifecycleEvidence{
		pass: pass, analyzer: analyzer, check: check,
		probe: analysisTrace.ForPackage(pass, analyzer, check),
	}
	evidence.local = lifecycle.NewLocalEvidenceWithReturnedCleanup(evidence.returnedCleanupLookup())
	return evidence
}

// ForCandidate attributes the evidence traced from here on to candidate, so a
// trace selector retrieves the whole proof built for it. Analyzers call this
// once before judging each candidate.
func (evidence *LifecycleEvidence) ForCandidate(candidate token.Pos) {
	evidence.probe = analysisTrace.For(evidence.pass, evidence.analyzer, evidence.check, candidate)
}

// ArgumentRetained reports whether the summary of the call's static callee
// marks the argument at index as retained. The second result is false when
// no summary is available, which callers must treat as unknown.
func (evidence *LifecycleEvidence) ArgumentRetained(instruction ssa.Instruction, index int) (bool, bool) {
	return evidence.CalleeClaims(instruction, index, ClaimRetains)
}

// EvidenceRequest describes local and imported relationships that can settle
// one lifecycle obligation. Completion and Transfer are independently
// optional; imported selectors are consulted only when local evidence does not
// prove the obligation.
type EvidenceRequest struct {
	Instruction ssa.Instruction
	Target      ssa.Value
	Completion  *lifecycle.CompletionRequest
	Transfer    *lifecycle.OwnershipTransferRequest
	Local       *proofs.Proof
	SelectMask  func(Fact) ParameterMask
	// StrictImportedProjection lets one analyzer map a summary parameter to an
	// exact, stable field/index path beneath its target. Ordinary fact matching
	// remains identity/containment-only.
	StrictImportedProjection bool
	ReceiverStore            bool
}

// Prove returns one lifecycle proof with explicit provenance. Missing imported
// summaries produce Unknown rather than being conflated with a disproved local
// relationship.
func (evidence *LifecycleEvidence) Prove(request EvidenceRequest) Proof {
	local := evidence.localProof(request)
	if local.Proven() {
		evidence.emit(request, local)
		return local
	}

	if abandonedSearch(local) {
		// The local walk was abandoned before it could decide, or found its
		// only completion inside a loop, so an imported summary that disproves
		// the release would turn a boundary the analysis gave up on into a
		// decision. Keep the undecided answer; a caller that must not report
		// on a guess checks for this reason.
		evidence.emit(request, local)
		return local
	}

	if imported, consulted := evidence.importedProof(request); consulted {
		evidence.emit(request, imported)
		return imported
	}
	evidence.emit(request, local)
	return local
}

func (evidence *LifecycleEvidence) importedProof(request EvidenceRequest) (Proof, bool) {
	fact, summarized := factFor(evidence.pass, request.Instruction)
	answer := decideLifecycleEvidence(importedEvidenceStage, evidence, request, fact, summarized)
	return answer.proof, answer.decided
}

func (evidence *LifecycleEvidence) selectedMaskProof(request EvidenceRequest, fact Fact) (Proof, bool) {
	answer := decideSelectedMaskEvidence(evidence, request, fact)
	return answer.proof, answer.decided
}

// argumentCaseCompletes reports whether a case of the callee's summary, one
// whose assumed Boolean arguments this call supplies as constants, completes
// the exact target on every normal return. Only the requested completion is
// matched; an argument case never answers a transfer or retention question.
func (evidence *LifecycleEvidence) argumentCaseCompletes(request EvidenceRequest, fact Fact) bool {
	if request.Completion == nil || len(fact.caseDischarges()) == 0 {
		return false
	}
	switch request.Instruction.(type) {
	case *ssa.Call, *ssa.Defer:
	default:
		return false
	}
	if request.Completion.InvokeTarget {
		query := suppliedCondition(request.Instruction, nil)
		return !query.Unconditional() && factArgumentMatches(request.Instruction, request.Target, conditionalMask(fact, "", true, query),
			func(argument, target ssa.Value) bool {
				return heapmodel.NewStorage(request.Completion.Budget).Same(argument, target).Proven()
			})
	}
	method := requestedMethod(request)
	return method != "" && fact.caseDischargesArgument(request.Instruction, request.Target, method, nil, evidence.probe.Observer())
}

func (evidence *LifecycleEvidence) localProof(request EvidenceRequest) Proof {
	return decideLifecycleEvidence(localEvidenceStage, evidence, request, Fact{}, false).proof
}

// abandonedSearch reports whether a local walk gave up before deciding: it ran
// out of budget, or found its only completion inside a loop.
func abandonedSearch(proof Proof) bool {
	return proof.Reason == proofs.EvidenceBudgetExhausted || proof.Reason == proofs.EvidenceCompletionInCycle
}

func importedProof(reason Reason, method string) Proof {
	return Proof{Proof: proofs.Proof{
		State: proofs.EvidenceProven, Method: method,
		Provenance: proofs.EvidenceFromImportedFact,
	}, SummaryReason: reason}
}

// question names what a request asked, so a trace tells apart the several
// answers one instruction gets: a release, a transfer, a proof the caller
// supplied, or a summary mask.
func (request EvidenceRequest) question() string {
	var parts []string
	if request.Local != nil {
		parts = append(parts, "local")
	}
	if request.Completion != nil {
		parts = append(parts, "release")
	}
	if request.Transfer != nil {
		parts = append(parts, "transfer")
	}
	if request.SelectMask != nil {
		parts = append(parts, "summary")
	}
	if request.ReceiverStore {
		parts = append(parts, "receiver-store")
	}
	return strings.Join(parts, "+")
}

func requestedMethod(request EvidenceRequest) string {
	if request.Completion == nil || len(request.Completion.Methods) != 1 {
		return ""
	}
	return request.Completion.Methods[0]
}

func (evidence *LifecycleEvidence) emit(request EvidenceRequest, proof Proof) {
	if !evidence.probe.Enabled() {
		return
	}
	outcome := analysisTrace.OutcomeUnknown
	switch proof.State {
	case proofs.EvidenceProven:
		outcome = analysisTrace.OutcomeAccepted
	case proofs.EvidenceDisproven:
		outcome = analysisTrace.OutcomeRejected
	case proofs.EvidenceUnknown:
	}
	details := evidenceDetails(request.Instruction, request.Target)
	details["question"] = request.question()
	if proof.Method != "" {
		details["method"] = proof.Method
	}
	if proof.Provenance != 0 {
		details["provenance"] = proof.Provenance.String()
	}
	position := token.NoPos
	if request.Instruction != nil {
		position = request.Instruction.Pos()
	}
	evidence.probe.Evidence(analysisTrace.Step{
		Reason:   proof.traceReason(),
		Outcome:  outcome,
		Pos:      position,
		Function: instructionFunction(request.Instruction),
		Details:  details,
	})
}

func evidenceDetails(instruction ssa.Instruction, target ssa.Value) map[string]string {
	details := map[string]string{}
	if instruction != nil {
		// The SSA text of the instruction lets a reader follow the trace as an
		// annotated SSA walk without rebuilding the lowering by hand.
		details["instruction"] = instruction.String()
	}
	if target != nil {
		details["target"] = target.Name()
		if target.Type() != nil {
			details["target_type"] = target.Type().String()
		}
	}
	if common := ssaflow.InstructionCall(instruction); common != nil && common.StaticCallee() != nil {
		details["callee"] = common.StaticCallee().String()
	}
	return details
}

func instructionFunction(instruction ssa.Instruction) string {
	if instruction == nil || instruction.Parent() == nil {
		return ""
	}
	return instruction.Parent().String()
}

// CallEffectsWithin exposes local call-effect evidence beside lifecycle evidence.
// Effects are possible uses, never proof of cleanup or ownership transfer;
// an absent Retained bit is not a read-only contract. Imported bodies stay unknown.
// Existing visits share budget while retaining the local QueryBudget cap. Cutoff
// is unknown even if the caller has allowance left. Graph/alias/type and binding
// construction costs remain separate. Nil selects an independent default allowance
// through the same authoritative trace path.
func (evidence *LifecycleEvidence) CallEffectsWithin(
	instruction ssa.Instruction, target ssa.Value, budget *proofs.SearchBudget,
) ssacall.CallEffectProof {
	queryBudget := budget.Within(proofs.QueryBudget).Observed(evidence.probe.Observer())
	proof := ssacall.NewCallEffects(queryBudget).Call(instruction, target)
	if !evidence.probe.Enabled() {
		return proof
	}
	details := evidenceDetails(instruction, target)
	for name, effect := range map[string]ssacall.CallEffect{
		"read": ssacall.EffectRead, "mutate": ssacall.EffectMutate,
		"retain": ssacall.EffectRetain, "async": ssacall.EffectAsync, "invoke": ssacall.EffectInvoke,
	} {
		details[name] = strconv.FormatBool(proof.Effects&effect != 0)
	}
	outcome := analysisTrace.OutcomeUnknown
	if proof.Proven() {
		outcome = analysisTrace.OutcomeAccepted
	}
	position := token.NoPos
	if instruction != nil {
		position = instruction.Pos()
	}
	evidence.probe.Evidence(analysisTrace.Step{
		Reason: proof.Reason.String(), Outcome: outcome, Pos: position,
		Function: instructionFunction(instruction), Details: details,
	})
	return proof
}

// evidencePolicyStage keeps local and imported source precedence together.
// Mask selection has its own generated entry point, so importing a mask never
// creates a recursive dispatcher call graph while this package is analyzed.
type evidencePolicyStage uint8

type evidenceProbe = analysisTrace.Probe

const (
	localEvidenceStage evidencePolicyStage = iota
	importedEvidenceStage
)

type evidenceAnswer struct {
	proof   Proof
	decided bool
}

type evidencePolicySignature func(evidencePolicyStage, *LifecycleEvidence, EvidenceRequest, Fact, bool) evidenceAnswer

type selectedMaskSignature func(*LifecycleEvidence, EvidenceRequest, Fact) evidenceAnswer

var (
	_ evidencePolicySignature = decideLifecycleEvidence
	_ selectedMaskSignature   = decideSelectedMaskEvidence
)

func suppliedEvidence(request EvidenceRequest) Proof {
	if request.Local != nil {
		return Proof{Proof: *request.Local}
	}
	return unavailableEvidence()
}

// Optional request binding stays native; each reached query still uses the
// original request and allowance. The rules own precedence between its answer
// and the other local/imported evidence.
func (evidence *LifecycleEvidence) completionEvidence(request EvidenceRequest, prior Proof) Proof {
	if request.Completion == nil {
		return prior
	}
	return Proof{Proof: evidence.local.Completion(*request.Completion).Proof}
}

func (evidence *LifecycleEvidence) capturedCompletionEvidence(request EvidenceRequest) Proof {
	if request.Completion == nil {
		return Proof{}
	}
	return evidence.capturedImportedCompletion(request)
}

func (evidence *LifecycleEvidence) transferEvidence(request EvidenceRequest) (Proof, bool) {
	if request.Transfer == nil {
		return Proof{}, false
	}
	return Proof{Proof: evidence.local.OwnershipTransfer(*request.Transfer).Proof}, true
}

func selectedEvidenceMask(request EvidenceRequest, fact Fact) ParameterMask {
	return request.SelectMask(fact)
}

func ordinaryEvidenceMask(fact Fact, mask ParameterMask, method string) ParameterMask {
	return mask &^ fact.MethodMask(method)
}

func directDischargeEvidence(fact Fact, request EvidenceRequest, method string, observer proofs.Observer) bool {
	return method != "" && fact.dischargesArgument(request.Instruction, request.Target, method, observer)
}

func projectedArgumentEvidence(request EvidenceRequest, mask ParameterMask, observer proofs.Observer) bool {
	return request.StrictImportedProjection && factOwnsProjectedArgument(request.Instruction, request.Target, mask, observer)
}

func possibleArgumentEvidence(request EvidenceRequest, mask ParameterMask) bool {
	return factArgumentMatches(request.Instruction, request.Target, mask, heapmodel.MayAlias)
}

func selectedEvidenceRequested(request EvidenceRequest) bool {
	return request.SelectMask != nil
}

func receiverStoreEvidence(evidence *LifecycleEvidence, request EvidenceRequest, fact Fact) bool {
	return request.ReceiverStore &&
		factOwnsArgument(request.Instruction, request.Target, fact.ReceiverStore(), evidence.probe.Observer())
}

func receiverEscapes(request EvidenceRequest) bool {
	receiver := ssaflow.CallReceiver(ssaflow.InstructionCall(request.Instruction))
	return receiver != nil && (ssaflow.ExternallyOwnedValue(receiver) || lifecycle.ValueHasTransferUse(receiver))
}

func importedEvidenceRequested(request EvidenceRequest) bool {
	return request.SelectMask != nil || request.ReceiverStore
}

func unavailableEvidence() Proof {
	return Proof{Proof: proofs.Proof{State: proofs.EvidenceUnknown, Reason: proofs.EvidenceUnavailable}}
}

func unmatchedImportedEvidence() Proof {
	return Proof{Proof: proofs.Proof{
		State: proofs.EvidenceDisproven, Reason: proofs.EvidenceNotFound,
		Provenance: proofs.EvidenceFromImportedFact,
	}}
}

func unescapedReceiverEvidence() Proof {
	return Proof{Proof: proofs.Proof{
		State: proofs.EvidenceDisproven, Provenance: proofs.EvidenceFromImportedFact,
	}, SummaryReason: reasonReceiverDoesNotEscape}
}

func emptyEvidence() Proof { return Proof{} }
