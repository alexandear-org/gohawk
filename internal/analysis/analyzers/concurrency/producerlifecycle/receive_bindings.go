package producerlifecycle

import (
	"go/token"
	"go/types"

	"github.com/kojah/gohawk/internal/analysis/passes/concurrencyfacts"
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	proofs "github.com/kojah/gohawk/internal/engine/proof"

	"golang.org/x/tools/go/ssa"
)

// Receive recognition asks the existing helper and alias proofs at the reached
// instruction. The caller retains block traversal, accumulation and cycle cuts.
type receiveSignature func(ssa.Instruction, ssa.Value, *ssa.Go, *concurrencyfacts.Engine, *proofs.SearchBudget) receiveProof

var _ receiveSignature = instructionReceives

const receiveToken = token.ARROW

func callInstruction(instruction ssa.Instruction) (ssa.CallInstruction, bool) { //nolint:ireturn // Exact call site.
	call, ok := instruction.(ssa.CallInstruction)
	return call, ok
}

func selectedReceives(selected *ssa.Select, channel ssa.Value) receiveProof {
	var proof receiveProof
	for _, state := range selected.States {
		if state.Dir == types.RecvOnly && heapmodel.MayAlias(state.Chan, channel) {
			proof.count++
		}
	}
	return proof
}

// Producer policies retain native requests and proof payloads. Their queries
// call the same summary, count and capture engines with no budget reset.
type (
	abandonmentSignature   func(*ssa.Function, producerSend, []producerSend, *concurrencyfacts.Engine) producerProof
	helperReceiveSignature func(ssa.CallInstruction, ssa.Value, *ssa.Go, *concurrencyfacts.Engine, *proofs.SearchBudget) receiveProof
)

var (
	_ abandonmentSignature   = abandonedProducerSend
	_ helperReceiveSignature = helperReceives
)

const (
	producerUnknown   = proofs.EvidenceUnknown
	producerProven    = proofs.EvidenceProven
	producerDisproven = proofs.EvidenceDisproven
)

func producerOrigin(call ssa.CallInstruction, origin *ssa.Go) bool { return call == origin }

func producerBuiltin(call ssa.CallInstruction) bool {
	_, builtin := call.Common().Value.(*ssa.Builtin)
	return builtin
}

func receiverSummary(engine *concurrencyfacts.Engine, call ssa.CallInstruction, budget *proofs.SearchBudget) concurrencyfacts.Summary {
	return engine.AtCall(call, budget)
}

func receiverComplete(summary concurrencyfacts.Summary) bool { return summary.Complete() }

func receiverExhausted(budget *proofs.SearchBudget) bool { return budget.Exhausted() }

func receiverLaunched(call ssa.CallInstruction) bool {
	_, launched := call.(*ssa.Go)
	return launched
}

func opaqueReceiver(uncertain bool) receiveProof {
	return receiveProof{unknown: uncertain, reason: reasonReceiverHelperUnknown}
}
