package processownership

import (
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/reporting/check"
	analysisTrace "github.com/kojah/gohawk/internal/reporting/trace"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ssa"
)

// This file owns final process decision presentation and post-Start policy.
// Pre-Start suppressions and post-Start results share the same trace projection.
// An unowned return is insufficient when the handle is unused: detached launch
// intent remains unknown. Tracing consumes the same decision as reporting.
type processDecision struct {
	state  proofs.EvidenceState
	reason processReason
}

func emitProcessDecision(pass *analysis.Pass, function *ssa.Function, start *ssa.Call, command ssa.Value, decision processDecision) {
	checkID := string(check.ProcessWait)
	if !analysisTrace.Enabled("processownership", checkID) {
		return
	}
	outcome := analysisTrace.DiagnosticOutcome(decision.state)
	details := map[string]string{}
	if command != nil && command.Type() != nil {
		details["command_type"] = command.Type().String()
	}
	analysisTrace.For(pass, "processownership", checkID, start.Pos()).Decision(analysisTrace.Step{
		Reason:   decision.reason.String(),
		Outcome:  outcome,
		Pos:      start.Pos(),
		Function: function.String(),
		Details:  details,
	})
}
