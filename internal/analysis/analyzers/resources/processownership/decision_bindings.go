package processownership

import (
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"golang.org/x/tools/go/ssa"
)

// The generated policy consumes the existing command-use proof and Boolean
// entry-lifetime evidence directly. Searches keep their original allowance.
type processReturnSignature func(start *ssa.Call, command ssa.Value, witness *ssa.Return, unknown bool, budget *proofs.SearchBudget) processDecision

var _ processReturnSignature = decideProcessReturn

// Hyrule's named struct domain is the original proof, without wrapping it.
type processUseEvidence = proofs.Proof

const (
	decisionUnknown   = proofs.EvidenceUnknown
	decisionDisproven = proofs.EvidenceDisproven
	decisionProven    = proofs.EvidenceProven
)

func entryLifetime(start *ssa.Call) bool {
	return ssacall.RunsOnceInProgramEntry(start)
}
