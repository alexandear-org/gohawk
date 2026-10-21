package processownership

import (
	"github.com/kojah/gohawk/internal/analysis/passes/lifecyclefacts"
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/lifecycle"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// processQueryBudget bounds one completion, transfer, or handoff question
// the proof asks; exhaustion is unknown evidence, never a wait or a leak.
const processQueryBudget = proofs.QueryBudget

// processPoolBudget shares one allowance across the candidate's completion,
// transfer, discovery and reachability requests. A hundred query allowances
// leave ordinary candidates room while interrupted evidence stays unknown.
// Heap, type, symbol and other independently owned query costs remain separate.
const processPoolBudget = 100 * processQueryBudget

// commandProof is the evidence context for one started command: the
// lifecycle evidence its questions go through and the pool they draw on.
type commandProof struct {
	evidence *lifecyclefacts.LifecycleEvidence
	pool     *proofs.SearchBudget
	actions  map[commandActionKey]proofs.EvidenceState
}

// budget draws one query's allowance from the candidate's pool so the
// give-ups inside it reach the trace and its requests share one allowance.
func (proof *commandProof) budget() *proofs.SearchBudget {
	return proof.pool.Within(processQueryBudget)
}

// abandoned reports a question the search gave up on before deciding. What
// that permits is decided where the question was asked: before Start it
// leaves ownership possibly registered, after Start it leaves the action
// unknown, and neither is a reason to report.
func abandoned(result lifecyclefacts.Proof) bool {
	return result.Reason == proofs.EvidenceBudgetExhausted
}

// The generated stages preserve early handoff cutoffs, cheap settlement, and
// late exact witnesses. The same cache surrounds the complete instruction action.
func processOwnershipAction(proof *commandProof, instruction ssa.Instruction, command ssa.Value) proofs.EvidenceState {
	state, decided := recognizeProcessAction(proof, instruction, command)
	if decided {
		return state
	}
	// Allocate the original completion allowance after early queries and before
	// cheap settlement; only the lifecycle evidence query may spend it.
	settled, fallback := recognizeProcessSettlement(proof, instruction, command, proof.budget())
	if settled {
		return proofs.EvidenceProven
	}
	return decideProcessRemainder(proof, instruction, command, fallback)
}

type (
	processActionPrefixSignature func(proof *commandProof, instruction ssa.Instruction, command ssa.Value) (proofs.EvidenceState, bool)
	processSettlementSignature   func(
		proof *commandProof, instruction ssa.Instruction, command ssa.Value, budget *proofs.SearchBudget,
	) (bool, proofs.EvidenceState)
	processRemainderSignature func(proof *commandProof, instruction ssa.Instruction, command ssa.Value, fallback proofs.EvidenceState) proofs.EvidenceState
)

var (
	_ processActionPrefixSignature = recognizeProcessAction
	_ processSettlementSignature   = recognizeProcessSettlement
	_ processRemainderSignature    = decideProcessRemainder
)

func (proof *commandProof) possibleWaitHandoff(instruction ssa.Instruction, command ssa.Value) bool {
	return provePossibleWaitHandoff(instruction, command, proof.budget()).State == proofs.EvidenceUnknown
}

func (proof *commandProof) deferredWait(instruction ssa.Instruction, command ssa.Value) proofs.Proof {
	return proofs.Proof{State: deferredClosureWaitsForCommand(instruction, command, proof.budget())}
}

// os.Process.Release explicitly relinquishes the parent's wait/reap
// obligation for deliberately detached daemons:
// https://github.com/drn/argus/blob/9b4bb7e71217e22557f72531909bf803354d3ab4/internal/daemon/client/autostart_fork.go#L41-L45
func releasedProcess(instruction ssa.Instruction, command ssa.Value) bool {
	common := ssaflow.InstructionCall(instruction)
	return ssacall.CallMatchesSymbol(common, syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "os", Receiver: "Process", Name: "Release"})) &&
		heapmodel.ValueDerivesFrom(ssaflow.CallReceiver(common), command)
}

func exitedProcess(instruction ssa.Instruction) bool {
	return ssacall.CallMatchesSymbol(ssaflow.InstructionCall(instruction), syntax.PackageFunction("os", "Exit"))
}

// commandOwnership forwards the original evidence request and exposes its
// provisional fallback. Later exact witnesses may still override that fallback.
func (proof *commandProof) commandOwnership(instruction ssa.Instruction, command ssa.Value, budget *proofs.SearchBudget) (bool, proofs.EvidenceState) {
	completion := lifecycle.CompletionRequest{
		Instruction: instruction, Target: command, Methods: []string{"Wait"}, Budget: budget,
	}
	// A command stored into a package variable, or into a map this function
	// does not own, belongs to that registry: another function looks it up and
	// waits on it, as resourcelifetime already treats a stored resource.
	// https://github.com/alphagov/router/blob/7cfa97b4548fdf02836ab9c01c7df853899af6cd/integration_tests/router_support.go#L119-L127
	transfer := lifecycle.OwnershipTransferRequest{
		Instruction: instruction,
		Value:       command,
		Modes: lifecycle.TransferStoredInField | lifecycle.TransferOwnerStoredInField |
			lifecycle.TransferCapturedByClosure | lifecycle.TransferCallResultStoredInField |
			lifecycle.TransferStoredInGlobal | lifecycle.TransferStoredInOwnedMap,
	}
	// A launched waiter owns reaping when every normal goroutine return waits,
	// including a nested defer. feint uses both direct and deferred background
	// waiters for deliberately longer-lived commands:
	// https://github.com/stephrobert/feint/blob/270aeb83c264ad109af885bb4e52f598265c5e1f/internal/cli/lifecycle.go#L183-L206
	// https://github.com/stephrobert/feint/blob/270aeb83c264ad109af885bb4e52f598265c5e1f/internal/core/machine/incus_watch.go#L51-L61
	// The caller asks this only after cheap Wait/Release witnesses fail. Keep
	// an abandoned search as a provisional fallback; later exact handle or
	// process-exit evidence may still prove settlement.

	result := proof.evidence.Prove(lifecyclefacts.EvidenceRequest{
		Instruction: instruction, Target: command, Completion: &completion, Transfer: &transfer,
		SelectMask: func(fact lifecyclefacts.Fact) lifecyclefacts.ParameterMask {
			return fact.ReturnedOwner() | fact.MethodMask("Wait")
		},
		ReceiverStore: true,
	})
	fallback := proofs.EvidenceDisproven
	if abandoned(result) {
		fallback = proofs.EvidenceUnknown
	}
	return result.Proven(), fallback
}

func (proof *commandProof) handleOwnership(instruction ssa.Instruction, command ssa.Value) proofs.Proof {
	return proofs.Proof{State: processHandleOwnershipAction(proof, instruction, command)}
}

func storesProcessHandleInExternalField(instruction ssa.Instruction, command ssa.Value) bool {
	store, ok := instruction.(*ssa.Store)
	if !ok || !osProcessDerivedFromCommand(store.Val, command) {
		return false
	}
	field, ok := store.Addr.(*ssa.FieldAddr)
	// Persisting the process handle on a caller-owned receiver transfers the
	// reaping obligation without exposing *exec.Cmd itself. GitHub CLI starts a
	// pager this way and waits from StopPager:
	// https://github.com/cli/cli/blob/d528f20f2ee02f6703773e9f56c90e3c3f5d46b0/pkg/iostreams/iostreams.go#L256-L274
	return ok && ssaflow.ExternallyOwnedValue(field.X)
}

func processHandleOwnershipAction(proof *commandProof, instruction ssa.Instruction, command ssa.Value) proofs.EvidenceState {
	if returned, ok := instruction.(*ssa.Return); ok && !returnsProcessHandle(returned, command) {
		// A returned aggregate can keep the started child's lower-level handle
		// without keeping exec.Cmd. Containment is a possible ownership handoff,
		// not proof that the owner will Wait; PID-only projections do not qualify.
		// https://github.com/criyle/go-sandbox/blob/6a60e40be9d0cefb656c4ae12415c5fd040df954/container/environment_linux.go#L266-L280
		if owner := proveReturnedProcessOwner(returned, command, proof.budget()); owner.State != proofs.EvidenceDisproven {
			return proofs.EvidenceUnknown
		}
		return proofs.EvidenceDisproven
	}
	common := ssaflow.InstructionCall(instruction)
	if common == nil {
		return proofs.EvidenceDisproven
	}
	// A helper handed the lower-level handle can reap the child without ever
	// seeing exec.Cmd. Bind its Wait or returned-owner summary to that exact
	// argument; incomplete searches remain unknown rather than absence of cleanup.
	state := proofs.EvidenceDisproven
	arguments := proof.budget()
	for _, argument := range common.Args {
		if !arguments.Spend() {
			return proofs.EvidenceUnknown
		}
		if !osProcessDerivedFromCommand(argument, command) {
			continue
		}
		completion := lifecycle.CompletionRequest{
			Instruction: instruction,
			Target:      argument,
			Methods:     []string{"Wait"},
			Budget:      proof.budget(),
		}
		result := proof.evidence.Prove(lifecyclefacts.EvidenceRequest{
			Instruction: instruction,
			Target:      argument,
			Completion:  &completion,
			SelectMask: func(fact lifecyclefacts.Fact) lifecyclefacts.ParameterMask {
				return fact.ReturnedOwner() | fact.MethodMask("Wait")
			},
		})
		if result.Proven() {
			return proofs.EvidenceProven
		}
		if abandoned(result) {
			state = proofs.EvidenceUnknown
		}
	}
	return state
}
