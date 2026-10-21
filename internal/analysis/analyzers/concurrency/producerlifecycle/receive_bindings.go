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
