package analyzers

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/kojah/gohawk/internal/analysis/catalog"
	"github.com/kojah/gohawk/internal/reporting/check"
	analysisTrace "github.com/kojah/gohawk/internal/reporting/trace"
	"golang.org/x/tools/go/analysis"
)

func TestDiagnosticAdmissionOrderAndReportingEffects(t *testing.T) {
	files, parsed, positions := admissionSource(t)

	production := positions["production.go/ordinary"]
	suppressedTest := positions["production_test.go/suppressed"]
	diagnostics := []analysis.Diagnostic{
		{Category: "example/old", Pos: suppressedTest},
		{Category: "example/missing", Pos: suppressedTest},
		{Category: "example/disabled", Pos: suppressedTest},
		{Category: "example/known", Pos: suppressedTest},
		{Category: "example/known", Pos: positions["production_test.go/ordinary"]},
		{Category: "example/known", Pos: production},
		{Category: "example/known", Pos: production, URL: "https://example.com/custom"},
		{Category: "example/other", Pos: production},
	}
	var reported []analysis.Diagnostic
	pass := &analysis.Pass{Fset: files, Files: parsed, Report: func(diagnostic analysis.Diagnostic) {
		reported = append(reported, diagnostic)
	}}
	var records []analysisTrace.Record
	restore := analysisTrace.Capture([]string{"example"}, "", func(record analysisTrace.Record) { records = append(records, record) })
	defer restore()
	originalError := errors.New("analyzer failed")
	originalResult := &struct{}{}
	analyzer := &analysis.Analyzer{Name: "example", Run: func(pass *analysis.Pass) (any, error) {
		for _, diagnostic := range diagnostics {
			pass.Report(diagnostic)
		}
		return originalResult, originalError
	}}
	wrapper := withCheckFilter(analyzer, []catalog.CheckInfo{{ID: "example/known"}, {ID: "example/disabled"}},
		[]check.ID{"example/old"}, map[string]bool{"example/old": true, "example/missing": true, "example/disabled": true}, "https://example.com/docs")
	result, err := wrapper.Run(pass)
	if result != originalResult || !errors.Is(err, originalError) {
		t.Fatalf("Run() = %v, %v, want original result and error", result, err)
	}
	for _, category := range []string{"example/missing", "example/other"} {
		if !strings.Contains(err.Error(), `reported unknown check "`+category+`"`) {
			t.Errorf("Run() error %q omits %q", err, category)
		}
	}
	if len(reported) != 2 || reported[0].URL != "https://example.com/docs" || reported[1].URL != "https://example.com/custom" {
		t.Fatalf("reported = %+v, want two admitted diagnostics with fallback/custom URLs", reported)
	}
	wantReasons := []check.ReportingReason{
		check.ReportingDelisted, check.ReportingUnknownCheck, check.ReportingNotSelected,
		check.ReportingSuppressed, check.ReportingTestFileSkipped, check.ReportingUnknownCheck,
	}
	if len(records) != len(wantReasons) {
		t.Fatalf("trace records = %+v, want %d decisions", records, len(wantReasons))
	}
	for index, reason := range wantReasons {
		wantOutcome := analysisTrace.OutcomeAccepted
		if reason == check.ReportingUnknownCheck {
			wantOutcome = analysisTrace.OutcomeRejected
		}
		if records[index].Reason != reason.String() || records[index].Outcome != wantOutcome || records[index].Phase != "decision" {
			t.Errorf("trace[%d] = %+v, want %v/%v decision", index, records[index], reason, wantOutcome)
		}
	}
	// The wrapper restores the host callback even when it joins report errors.
	pass.Report(analysis.Diagnostic{Category: "after-run"})
	if len(reported) != 3 || reported[2].Category != "after-run" || len(records) != len(wantReasons) {
		t.Fatal("Run left its reporting or tracing wrapper installed")
	}
}

func TestDiagnosticAdmissionStopsBeforeSourceQueries(t *testing.T) {
	// Each early outcome deliberately overlaps later filters and must be
	// decided even without a pass: source queries are not reached.
	for _, scenario := range []struct {
		name      string
		declared  map[string]bool
		withdrawn map[string]bool
		want      check.ReportingReason
	}{
		{"withdrawn", nil, map[string]bool{"example/check": true}, check.ReportingDelisted},
		{"unknown", nil, nil, check.ReportingUnknownCheck},
		{"disabled", map[string]bool{"example/check": true}, nil, check.ReportingNotSelected},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			got := check.DiagnosticAdmission(nil, analysis.Diagnostic{Category: "example/check"}, "example",
				scenario.declared, scenario.withdrawn, map[string]bool{"example/check": true})
			if got != scenario.want {
				t.Fatalf("admission = %v, want %v", got, scenario.want)
			}
		})
	}
}

func admissionSource(t *testing.T) (*token.FileSet, []*ast.File, map[string]token.Pos) {
	t.Helper()
	files := token.NewFileSet()
	positions := make(map[string]token.Pos)
	var parsed []*ast.File
	for _, name := range []string{"production.go", "production_test.go"} {
		file, err := parser.ParseFile(files, name, `package example
// gohawk:ignore example
func suppressed() {}
func ordinary() {}
`, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		parsed = append(parsed, file)
		for _, declaration := range file.Decls {
			function := declaration.(*ast.FuncDecl)
			positions[name+"/"+function.Name.Name] = function.Pos()
		}
	}

	return files, parsed, positions
}
