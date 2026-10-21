# Hyrule migration completion audit

The objective is to use Hyrule wherever it is a strong fit across GoHawk.
Completion remains unproven. A growing target count measures integration work,
not the fraction of suitable policy that has been reviewed.

## Completion requirements

| Requirement | Evidence needed | Current evidence |
| --- | --- | --- |
| Review the whole repository | Source inventory including small functions and code outside `internal` | Baseline inventory below; internal review remains open |
| Migrate strong fits | Production consumers call generated policies and the former decision path is removed | `hyrule.yaml` and consumer diffs; integration guide maps the migrated families |
| Keep the migration useful | Review ordered recognizers, evidence decisions and state policies; distinguish traversal and effect application | Bounded scope findings below; a branch-count threshold cannot establish this |
| Preserve semantics | Focused boundary tests, query-order checks, relevant trace/JSON checks and repository gates | Per-batch validation; new admission tests cover precedence and reporting effects |
| Preserve the development setup | On-disk compiler symlink, local workspace, ignored generated sources, CI generation | Integration guide and build configuration; no runtime rule interpretation |

## Inventory baseline

The source inventory at `83700fad` contains 363 authored non-test Go files and
2,632 function/method bodies: 345 files and 2,559 bodies under `internal`, and
18 files and 73 bodies elsewhere. Files with no function bodies are included
in the file total. Testdata, generated Go, dependency trees and hidden build
directories are excluded. Documentation examples are included for review, but
are examples of analyzed Go, not analyzer implementation to translate.

The earlier scan selected only functions with at least four switch cases or
six `if` statements under `internal`. That was a candidate finder, not a
completion audit. The expanded inventory has no branching threshold. Its
baseline agrees with the graph's 18 non-internal authored Go paths.

Graph verification used project `home-james-scratch-gohawk`, generation
`2026-10-21T23:25:21Z`, with complete bounded query results and exact-source
reads. A follow-up at `2026-10-21T23:27:07Z` checked every non-internal path.
Those paths had no recorded coverage issues. The broader tools scope excludes
two generated protobuf fixtures and the compiler symlink; their generated
headers and symlink target were checked directly. Clean path metadata
does not prove source completeness. Graph method-call resolution can confuse
same-named methods, so source is authoritative for query ordering and identity.
Rules are read directly because the graph does not index their language.

## Reviewed non-internal scope

All 18 non-internal Go paths were read, including functions below the former
branch-count threshold. One missed strong fit was found in diagnostic admission.

| Files | Finding |
| --- | --- |
| `analyzers/analyzers.go` | `withCheckFilter` had five ordered admission decisions and repeated trace construction. `check.DiagnosticAdmission` now selects the reason through rules; the wrapper applies that decision. Remaining functions build catalog views, forward calls, construct wrappers or collect timing. |
| `analyzers/catalog_specs.go`, `checks.go`, `doc.go` | Catalog declarations already form a data table; type aliases, metadata forwarding and package documentation need no rule layer. |
| `plugin/golangci/golangci.go` | Plugin registration, settings decoding, selection-map construction, validation and analyzer wrapping. Selection precedence is applied while accumulating configuration; the small final Boolean selection does not justify another adapter. |
| `tools/gendocs/blocks.go` | Marker slicing, frontmatter parsing/validation, escaping and rendering. Failures carry specific errors and offsets; this is a text-processing boundary rather than a structural recognizer or evidence policy. |
| `tools/gendocs/development.go`, `helpers.go` | Filesystem and declaration collection, Go parser/doc/printer APIs, reference rendering and comparison. Small AST filters stay within those collection loops. |
| `tools/gendocs/main.go`, `synchronize.go` | Flag handling, generation orchestration, page validation, filesystem synchronization and error propagation. |
| `tools/gendocs/manifest.go`, `index.go`, `timing.go` | Catalog projection/validation, HTML/Markdown rendering, sorting and measurements. Existing declaration tables remain authoritative. |
| `tools/measure/main.go` | Process execution, resource-usage reflection, platform unit conversion and measurement output; no substantial policy family to extract. |
| `main.go`, `generate.go` | Executable exit boundary and generation directives. |
| `tools/gendocs/ssaexample/example.go`, `site/examples/lock-analysis/example.go` | Illustrative Go inputs whose exact lowering or defects are the subject of documentation. |

### Diagnostic admission contract

`internal/reporting/check/report_admission.rules` preserves this exact order:
withdrawn category, unknown category, disabled category, adjacent suppression,
test-file backstop, admission. Withdrawn categories remain recognized even when
absent from the public catalog. An unknown category remains an error even if
disabled. Suppression wins over the test-file reason when both apply.

Source queries run only when reached. The wrapper still emits the existing
reason/outcome for a filtered diagnostic, accumulates unknown-category errors,
supplies a missing URL, preserves a custom URL, restores `pass.Report`, and
returns the analyzer's original result/error. Admission itself emits no event;
the outer reporting filter retains ownership of the emitted-diagnostic event.
`analyzers/reporting_test.go` exercises overlapping filters, early exits without
a source pass, trace polarity, multiple report errors and callback restoration.

## Reviewed internal infrastructure scopes

The authored production Go in `internal/reporting` and
`internal/analysis/catalog` was read in full. The graph returned all 24 paths
in these scopes, including tests. The only recorded scope gap was the ignored
generated admission function, which was inspected directly.

| Scope | Disposition |
| --- | --- |
| `reporting/check` | Admission is the substantial ordered policy and now uses rules. The separate plugin filter has one disabled-category branch and owns callback installation/restoration. Suppression performs a comment census. Evidence helpers construct ranges and walk dominance; their one- or two-form leaf matches do not justify independent adapters. Reporting identities and reason rendering are mechanical declarations. |
| `reporting/trace` | Event filtering uses compact selector predicates; phase/outcome mappings are small enum operations. The remaining implementation owns global configuration, locking, writer/capture lifetimes, timing and JSONL serialization. Translating its option switch would retain the same I/O and synchronization in query hooks. |
| `analysis/catalog` | Declaration validation accumulates indexed groups/check owners and produces contextual errors. Construction, withdrawal filtering, ordering and projection stay in Go. Tier/kind behavior is already a small table. Cloning detaches check/withdrawn lists while preserving analyzer pointers; the few slice operations do not warrant a schema maintenance layer. |

Ten further authored files in `engine/proof`, `engine/enumtext` and
`analysis/factcodec` were read in full, with complete graph path results and
clean coverage metadata at `2026-10-21T23:30:18Z`.

| Scope | Disposition |
| --- | --- |
| `engine/proof` | Evidence vocabulary and reason tables, allowance charging, parent-pool propagation, observation and recorder lifetimes. The package deliberately does not decide proof polarity; its small availability checks and state mutation remain the query substrate. |
| `engine/enumtext` | Generic numeric-label lookup and encoding/decoding over caller-supplied tables. Already declarative; no domain recognizer or ordered evidence decision remains. |
| `analysis/factcodec` | CBOR envelope validation, encoding/decoding, immutable payload publication and a shared `sync.Once` encoding cache. Boundary checks protect the wire format; moving them to query hooks would retain serialization and cache ownership in Go without reducing a policy family. |

All six authored files in `engine/syntax` were also read, with complete graph
results and clean path/scope metadata at the same generation. Source-range
selection, enclosing-node searches and wrapper peeling are AST traversal.
Assignment extraction has two parent forms; type helpers ask narrowly defined
`go/types` questions. `Symbol.MatchesObject` dispatches four explicit symbol
kinds to the existing package/receiver identity helpers after one exact-name
check. The remaining native identity operations would still be required behind
rule hooks, so this compact dispatch stays in Go. Source-file admission similarly
combines generated/test-file checks with driver/variant queries; it has no
substantial independent recognition family to extract.

The two authored files in `analysis/summaries` were read in full, alongside
their condition/comparison rules. Those generated policies already own result
condition recognition and comparison order. The broker's remaining code selects
prerequisites, exposes availability, binds summary slots, and supplies bounded
flow hooks. `pairedNilness` iterates published cases and asks dominating-path
evidence; its final outcome switch only interprets nil/non-nil enum values.
These operations retain their native Go orchestration. Exact path metadata was
clean; the scope's only graph gaps were the two ignored generated outputs,
confirmed as generated files and covered by the generation check.

## Remaining audit work

The internal scope still needs a complete source review with per-family
dispositions. Earlier migration work and the integration guide establish many
positive integrations, but do not establish that every remaining strong fit
has been found. In particular:

- Review smaller recognizers and remaining mixed policy/traversal functions.
- Resolve conditional candidates against existing generated policies before
  introducing new interfaces. Parent cancellation, for example, requires result
  zero and same-function provenance; the context-origin recognizer admits a
  different result contract and cannot be substituted without proof.
- Review remaining schema-copy/equality candidates against their actual mutable
  ownership contracts. A generic deep copy is not equivalent to intentional
  borrowing or branch-local detachment.
- Check the non-Go tooling/configuration inventory for any relevant integration
  gaps; the Go inventory alone cannot establish whole-repository completion.
- After remaining migrations, verify consumers and removed decision paths,
  generation, relevant behavior and the final repository gate. Reuse unchanged
  validation rather than repeatedly running a broad gate during discovery.

The audit should close these scopes before claiming that the goal is complete.
