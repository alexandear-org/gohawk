package trace

import (
	"bytes"
	"encoding/json"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestEmitSelectsByCandidateNotEventPosition(t *testing.T) {
	resetTrace(t)
	var output bytes.Buffer
	global.config.writer = &output
	global.config.selectors = map[string]bool{"resourcelifetime": true}
	global.config.candidate = "resource.go:7"
	global.active.Store(true)

	files := token.NewFileSet()
	file := files.AddFile("resource.go", -1, 100)
	file.SetLines([]int{0, 10, 20, 30, 40, 50, 60})
	callee := files.AddFile("callee.go", -1, 100)
	callee.SetLines([]int{0, 10, 20})
	pass := &analysis.Pass{Fset: files}

	// An analyzer selector still excludes another analyzer's events.
	For(pass, "goroutineownership", "", file.Pos(60)).Decision(Step{Reason: "ignored"})
	// A step about a callee body in another file belongs to the candidate under
	// study, so it survives selection even though its own position does not match.
	For(pass, "resourcelifetime", "resourcelifetime/missing-release", file.Pos(60)).Evidence(Step{
		Reason:  "evidence-unavailable",
		Outcome: OutcomeUnknown,
		Pos:     callee.Pos(10),
		Details: map[string]string{"resource": "*os.File"},
	})
	// A step at the selected position that serves a different candidate does not.
	For(pass, "resourcelifetime", "", callee.Pos(10)).Decision(Step{Reason: "other-candidate", Pos: file.Pos(60)})

	if lines := bytes.Count(output.Bytes(), []byte("\n")); lines != 1 {
		t.Fatalf("expected one selected record, got %d: %s", lines, output.String())
	}
	var got Record
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &got); err != nil {
		t.Fatalf("decode trace: %v\n%s", err, output.String())
	}
	if got.Reason != "evidence-unavailable" || got.Candidate != "resource.go:7:1" ||
		got.Position != "callee.go:2:1" || got.Details["resource"] != "*os.File" {
		t.Fatalf("trace = %+v", got)
	}
}

func TestEmitSerializesConcurrentEvents(t *testing.T) {
	resetTrace(t)
	var output bytes.Buffer
	global.config.writer = &output
	global.config.selectors = map[string]bool{"all": true}
	global.active.Store(true)
	pass := &analysis.Pass{Fset: token.NewFileSet()}
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			For(pass, "test", "", token.NoPos).Decision(Step{Reason: "concurrent", Outcome: OutcomeObserved})
		})
	}
	workers.Wait()
	lines := bytes.Split(bytes.TrimSpace(output.Bytes()), []byte("\n"))
	if len(lines) != 20 {
		t.Fatalf("trace line count = %d, want 20", len(lines))
	}
	for _, line := range lines {
		if !json.Valid(line) {
			t.Fatalf("invalid trace line %q", line)
		}
	}
}

func TestRegisterFlagsRejectsInvalidValues(t *testing.T) {
	resetTrace(t)
	flags := flag.NewFlagSet("trace", flag.ContinueOnError)
	RegisterFlags(flags)
	if err := flags.Parse([]string{"-gohawk-trace=", "./..."}); err == nil {
		t.Fatal("empty trace selector was accepted")
	}
}

func TestEnabledUsesAnalyzerAndCheckSelectors(t *testing.T) {
	resetTrace(t)
	global.config.selectors = map[string]bool{"goroutineownership/unjoined": true}
	global.active.Store(true)
	if !Enabled("goroutineownership", "goroutineownership/unjoined") {
		t.Fatal("selected check is not enabled")
	}
	if Enabled("processownership", "processownership/missing-wait") {
		t.Fatal("unselected check is enabled")
	}
}

func TestEmitDiagnosticResolvesEnclosingFunction(t *testing.T) {
	resetTrace(t)
	var output bytes.Buffer
	global.config.writer = &output
	global.config.selectors = map[string]bool{"concurrentcapture": true}
	global.active.Store(true)

	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "sample.go", "package sample\nfunc openFile() { println() }\n", 0)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	function := file.Decls[0]
	pass := &analysis.Pass{Fset: files, Files: []*ast.File{file}}
	EmitDiagnostic(
		pass,
		DiagnosticEvent{
			Analyzer:   "concurrentcapture",
			Phase:      PhaseCandidate,
			Reason:     "diagnostic-candidate",
			Outcome:    OutcomeObserved,
			Diagnostic: analysis.Diagnostic{Category: "concurrentcapture/shared-capture", Pos: function.Pos(), Message: "unsafe send"},
		},
	)

	var got Record
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &got); err != nil {
		t.Fatalf("decode trace: %v\n%s", err, output.String())
	}
	if got.Function != "openFile" || got.Details["message"] != "unsafe send" {
		t.Fatalf("trace = %+v", got)
	}
}

func resetTrace(t *testing.T) {
	t.Helper()
	global.Lock()
	previous, previousTiming := global.config, timingActive.Load()
	global.config = settings{writer: osStderrForTest{}}
	global.active.Store(false)
	timingActive.Store(false)
	global.Unlock()
	t.Cleanup(func() {
		if err := Close(); err != nil {
			t.Error(err)
		}
		global.Lock()
		global.config = previous
		global.active.Store(len(previous.selectors) > 0)
		timingActive.Store(previousTiming)
		global.Unlock()
	})
}

type osStderrForTest struct{}

func (osStderrForTest) Write(value []byte) (int, error) { return len(value), nil }

func TestTimingFileRecordsOneLinePerRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "timing.jsonl")
	resetTrace(t)
	flags := flag.NewFlagSet("timing", flag.ContinueOnError)
	RegisterFlags(flags)
	if err := flags.Parse([]string{"-gohawk-timing-file=" + path}); err != nil {
		t.Fatal(err)
	}
	if !TimingEnabled() {
		t.Fatal("timing should be enabled after the flag is set")
	}
	RecordTiming(Timing{Package: "example.com/p", Analyzer: "resourcelifetime", DurationNS: 42, AllocBytes: 1024})
	RecordTiming(Timing{Package: "example.com/q", Analyzer: "lockorder", DurationNS: 7, AllocBytes: 0})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("timing file has %d lines, want 2:\n%s", len(lines), data)
	}
	var first Timing
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if first.Analyzer != "resourcelifetime" || first.DurationNS != 42 || first.AllocBytes != 1024 {
		t.Fatalf("first Record = %+v", first)
	}
}

func TestObserveEmitsGiveUpAsUnknownEvidence(t *testing.T) {
	resetTrace(t)
	var output bytes.Buffer
	global.config.writer = &output
	global.config.selectors = map[string]bool{"resourcelifetime": true}
	global.active.Store(true)

	files := token.NewFileSet()
	file := files.AddFile("resource.go", -1, 100)
	file.SetLines([]int{0, 10, 20})
	pass := &analysis.Pass{Fset: files}

	disabled := For(pass, "goroutineownership", "", file.Pos(10))
	if disabled.Observer() != nil {
		t.Fatal("a disabled probe must hand out no observer, so a silent budget builds no details")
	}
	probe := For(pass, "resourcelifetime", "resourcelifetime/missing-release", file.Pos(10))
	observer := probe.Observer()
	if observer == nil {
		t.Fatal("an enabled probe must hand out its observer")
	}
	observer("storage-address-escapes", file.Pos(20), map[string]string{"instruction": "sink(t0)"})

	var got Record
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &got); err != nil {
		t.Fatalf("decode trace: %v\n%s", err, output.String())
	}
	if got.Phase != "evidence" || got.Outcome != OutcomeUnknown || got.Reason != "storage-address-escapes" ||
		got.Candidate != "resource.go:2:1" || got.Position != "resource.go:3:1" || got.Details["instruction"] != "sink(t0)" {
		t.Fatalf("trace = %+v", got)
	}
}

func TestCaptureHandsRecordsToTheSinkAndRestores(t *testing.T) {
	resetTrace(t)
	var output bytes.Buffer
	global.config.writer = &output

	files := token.NewFileSet()
	file := files.AddFile("resource.go", -1, 100)
	file.SetLines([]int{0, 10, 20})
	pass := &analysis.Pass{Fset: files}

	var captured []Record
	restore := Capture([]string{"lockorder"}, "resource.go:2", func(record Record) { captured = append(captured, record) })
	For(pass, "lockorder", "", file.Pos(10)).Decision(Step{Reason: "kept"})
	For(pass, "lockorder", "", file.Pos(20)).Decision(Step{Reason: "other-candidate"})
	For(pass, "resourcelifetime", "", file.Pos(10)).Decision(Step{Reason: "other-analyzer"})
	restore()
	For(pass, "lockorder", "", file.Pos(10)).Decision(Step{Reason: "after-restore"})

	if len(captured) != 1 || captured[0].Reason != "kept" || captured[0].Candidate != "resource.go:2:1" {
		t.Fatalf("captured = %+v, want only the kept step", captured)
	}
	if output.Len() != 0 || global.active.Load() {
		t.Fatalf("restore left tracing active or wrote %q", output.String())
	}
}

func TestNativeSourcePathsMatchPortableCandidateFilters(t *testing.T) {
	resetTrace(t)
	files := token.NewFileSet()
	file := files.AddFile(filepath.Join("source", "resource.go"), -1, 100)
	file.SetLines([]int{0, 10, 20})
	pass := &analysis.Pass{Fset: files}
	for _, candidate := range []string{"source/resource.go:2", filepath.Join("source", "resource.go") + ":2"} {
		var records []Record
		restore := Capture([]string{"test"}, candidate, func(record Record) { records = append(records, record) })
		probe := For(pass, "test", "", file.Pos(10))
		if !probe.Enabled() {
			t.Errorf("candidate %q did not select native source path", candidate)
		}
		probe.Decision(Step{Reason: "portable-position", Pos: file.Pos(20)})
		restore()
		if len(records) != 1 || records[0].Candidate != "source/resource.go:2:1" || records[0].Position != "source/resource.go:3:1" {
			t.Errorf("candidate %q: records = %+v", candidate, records)
		}
	}
}

func TestCloseReleasesBothDestinations(t *testing.T) {
	directory := t.TempDir()
	resetTrace(t)
	flags := flag.NewFlagSet("destinations", flag.ContinueOnError)
	RegisterFlags(flags)
	for name, value := range map[string]string{
		"gohawk-trace": "all", "gohawk-trace-file": filepath.Join(directory, "trace.jsonl"),
		"gohawk-timing-file": filepath.Join(directory, "timing.jsonl"),
	} {
		if err := flags.Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	traceFile, timingFile := global.config.file, global.config.timing
	if err := Close(); err != nil {
		t.Fatal(err)
	}
	for _, file := range []*os.File{traceFile, timingFile} {
		if _, err := file.Stat(); err == nil {
			t.Error("destination still open")
		}
		if err := os.Remove(file.Name()); err != nil {
			t.Errorf("remove closed destination: %v", err)
		}
	}
	if Enabled("test", "") || TimingEnabled() {
		t.Error("closing destinations left recording enabled")
	}
	if err := Close(); err != nil {
		t.Errorf("closing twice: %v", err)
	}
}
