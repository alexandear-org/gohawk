package check

import "golang.org/x/tools/go/analysis"

// Diagnostic admission selects the first reporting boundary that applies. The
// caller owns trace emission, unknown-category errors and publishing admitted
// diagnostics. An admitted result does not itself emit a reporting event.
type admissionSignature func(
	pass *analysis.Pass, diagnostic analysis.Diagnostic, analyzer string,
	declared, withdrawn, disabled map[string]bool,
) ReportingReason

var _ admissionSignature = DiagnosticAdmission

func categoryIncluded(categories map[string]bool, category string) bool {
	return categories[category]
}
