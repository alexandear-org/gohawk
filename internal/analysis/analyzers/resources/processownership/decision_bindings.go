package processownership

import (
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"golang.org/x/tools/go/ssa"
)

// These bindings expose post-Start evidence without choosing a diagnostic.
// The generated decision owns query order and uncertainty policy; SSA searches
// retain their existing budget and feasible-path semantics.
type processReturnRequest struct {
	start   *ssa.Call
	command ssa.Value
	witness *ssa.Return
	unknown bool
	budget  *proofs.SearchBudget
}

type processReturnSignature func(processReturnRequest) (processDecision, bool)

var _ processReturnSignature = processReturnDecision

type (
	processUseEvidence   struct{ state proofs.EvidenceState }
	processEntryEvidence struct{ owned bool }
)

const (
	decisionUnknown        = proofs.EvidenceUnknown
	decisionDisproven      = proofs.EvidenceDisproven
	decisionProven         = proofs.EvidenceProven
	entryOwned        bool = true
	entryOrdinary     bool = false
)

func commandUse(request processReturnRequest) processUseEvidence {
	proof := proveCommandUseAfterStart(request.start, request.command, request.budget)
	return processUseEvidence{proof.State}
}

func entryLifetime(request processReturnRequest) processEntryEvidence {
	return processEntryEvidence{ssacall.RunsOnceInProgramEntry(request.start)}
}
