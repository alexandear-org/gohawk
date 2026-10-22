// Command profilesummary aggregates the per-unit records gohawk writes under
// GOHAWK_PROFILE_DIR into a Markdown report of where a run spent its time.
//
// Each record times one go vet unit. Unit times are process wall time, and
// units run in parallel, so totals are summed work, not elapsed time. CPU
// profiles beside the records carry an "analyzer" label for pprof -tagfocus.
package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/kojah/gohawk/internal/cli/vetprofile"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, output, errorsOutput io.Writer) int {
	flags := flag.NewFlagSet("profilesummary", flag.ContinueOnError)
	flags.SetOutput(errorsOutput)
	top := flags.Int("top", 15, "rows in each per-unit table")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(errorsOutput, "usage: profilesummary [-top N] PROFILE_DIRECTORY")
		return 2
	}
	records, err := readRecords(flags.Arg(0))
	if err == nil && len(records) == 0 {
		err = fmt.Errorf("no records in %s", flags.Arg(0))
	}
	if err == nil {
		var report strings.Builder
		writeReport(&report, records, *top)
		_, err = io.WriteString(output, report.String())
	}
	if err != nil {
		_, _ = fmt.Fprintf(errorsOutput, "profilesummary: %v\n", err)
		return 1
	}
	return 0
}

func readRecords(directory string) ([]vetprofile.Record, error) {
	paths, err := filepath.Glob(filepath.Join(directory, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}
	records := make([]vetprofile.Record, 0, len(paths))
	var errs []error
	for _, path := range paths {
		data, err := os.ReadFile(path) //nolint:gosec // Records come from the directory the user named.
		if err != nil {
			errs = append(errs, fmt.Errorf("read record: %w", err))
			continue
		}
		var record vetprofile.Record
		if err := json.Unmarshal(data, &record); err != nil {
			errs = append(errs, fmt.Errorf("decode record %s: %w", path, err))
			continue
		}
		records = append(records, record)
	}
	return records, errors.Join(errs...)
}

type modeTotals struct {
	units    int
	setup    float64
	analysis float64
}

type analyzerTotal struct {
	name    string
	seconds float64
	units   int
	slowest float64
	where   string
}

func writeReport(output *strings.Builder, records []vetprofile.Record, top int) {
	var full, factsOnly modeTotals
	analyzers := make(map[string]*analyzerTotal)
	runsPerID := make(map[string]int)
	for _, record := range records {
		totals := &full
		if record.FactsOnly {
			totals = &factsOnly
		}
		totals.units++
		totals.setup += record.SetupSeconds
		totals.analysis += record.AnalysisSeconds
		runsPerID[record.ID]++
		for name, seconds := range record.Analyzers {
			total := analyzers[name]
			if total == nil {
				total = &analyzerTotal{name: name}
				analyzers[name] = total
			}
			total.seconds += seconds
			total.units++
			if seconds > total.slowest {
				total.slowest, total.where = seconds, record.Package
			}
		}
	}
	repeated := 0
	for _, runs := range runsPerID {
		if runs > 1 {
			repeated++
		}
	}

	_, _ = fmt.Fprintf(output, "# gohawk profile summary\n\n")
	_, _ = fmt.Fprintf(output, "Unit times are per-process wall seconds summed across parallel units: total work, not elapsed time.\n")
	_, _ = fmt.Fprintf(output, "Setup is parsing, type checking and fact decoding before the first analyzer runs.\n\n")
	_, _ = fmt.Fprintf(output, "- Units: %d (%d full, %d facts-only)\n", len(records), full.units, factsOnly.units)
	_, _ = fmt.Fprintf(output, "- Unit IDs analyzed more than once: %d\n\n", repeated)
	_, _ = fmt.Fprintf(output, "| Mode | Units | Setup (s) | Analysis (s) | Total (s) |\n|---|---:|---:|---:|---:|\n")
	for _, row := range []struct {
		name   string
		totals modeTotals
	}{{"full", full}, {"facts-only", factsOnly}} {
		_, _ = fmt.Fprintf(output, "| %s | %d | %.1f | %.1f | %.1f |\n",
			row.name, row.totals.units, row.totals.setup, row.totals.analysis, row.totals.setup+row.totals.analysis)
	}

	analysisTotal := full.analysis + factsOnly.analysis
	ranked := make([]*analyzerTotal, 0, len(analyzers))
	for _, total := range analyzers {
		ranked = append(ranked, total)
	}
	slices.SortFunc(ranked, func(a, b *analyzerTotal) int { return cmp.Compare(b.seconds, a.seconds) })
	_, _ = fmt.Fprintf(output, "\n## Analyzers\n\nAn analyzer's time is its own Run, excluding prerequisites. ")
	_, _ = fmt.Fprintf(output, "Analyzers overlap within a unit, so shares can sum past 100%%.\n\n")
	_, _ = fmt.Fprintf(output, "| Analyzer | Seconds | Share of analysis | Units | Slowest unit (s) | Slowest package |\n")
	_, _ = fmt.Fprintf(output, "|---|---:|---:|---:|---:|---|\n")
	for _, total := range ranked {
		_, _ = fmt.Fprintf(output, "| %s | %.2f | %.1f%% | %d | %.2f | `%s` |\n",
			total.name, total.seconds, percent(total.seconds, analysisTotal), total.units, total.slowest, total.where)
	}

	writeUnits(output, "Slowest units", records, top, func(a, b vetprofile.Record) int {
		return cmp.Compare(b.SetupSeconds+b.AnalysisSeconds, a.SetupSeconds+a.AnalysisSeconds)
	})
	writeUnits(output, "Largest units by peak RSS", records, top, func(a, b vetprofile.Record) int {
		return cmp.Compare(rss(b), rss(a))
	})
}

func writeUnits(output *strings.Builder, title string, records []vetprofile.Record, top int, order func(a, b vetprofile.Record) int) {
	sorted := slices.Clone(records)
	slices.SortFunc(sorted, order)
	_, _ = fmt.Fprintf(output, "\n## %s\n\n", title)
	_, _ = fmt.Fprintf(output, "| Package | Mode | Setup (s) | Analysis (s) | Peak RSS (MiB) | Heaviest analyzer |\n")
	_, _ = fmt.Fprintf(output, "|---|---|---:|---:|---:|---|\n")
	for _, record := range sorted[:min(top, len(sorted))] {
		mode := "full"
		if record.FactsOnly {
			mode = "facts-only"
		}
		_, _ = fmt.Fprintf(output, "| `%s` | %s | %.2f | %.2f | %s | %s |\n",
			record.Package, mode, record.SetupSeconds, record.AnalysisSeconds, formatRSS(record), heaviest(record))
	}
}

func heaviest(record vetprofile.Record) string {
	name, seconds := "", -1.0
	for candidate, elapsed := range record.Analyzers {
		if elapsed > seconds || (elapsed == seconds && candidate < name) {
			name, seconds = candidate, elapsed
		}
	}
	if name == "" {
		return "none"
	}
	return fmt.Sprintf("%s %.2fs", name, seconds)
}

// rss orders unknown peaks last; formatRSS prints them as unknown, never zero.
func rss(record vetprofile.Record) int64 {
	if record.PeakRSSKiB == nil {
		return -1
	}
	return *record.PeakRSSKiB
}

func formatRSS(record vetprofile.Record) string {
	if record.PeakRSSKiB == nil {
		return "unknown"
	}
	return strconv.FormatFloat(float64(*record.PeakRSSKiB)/1024, 'f', 1, 64)
}

func percent(part, whole float64) float64 {
	if whole == 0 {
		return 0
	}
	return 100 * part / whole
}
