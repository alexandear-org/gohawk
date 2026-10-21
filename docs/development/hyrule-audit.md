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

## Resource and call infrastructure review

The next bounded review read all 29 authored Go files in `engine/resourcemodel`,
`engine/ssaflow/cfg`, `engine/ssaflow/calls` (including `synchronization`) and
`analysis/passes/testvariant`. Complete graph path queries and exact-path checks
used generation `2026-10-21T23:33:45Z`. Authored paths had clean metadata; the
five scope gaps were ignored generated call/synchronization outputs. Their
headers and all five source rule files were inspected directly.

| Files or family | Disposition |
| --- | --- |
| `resourcemodel/state.go`, `relations.go` | Three independent obligation flags have simple immutable setters and one conjunction. Relation proofs sequence identity and stored-path queries with allowance fences, then construct an owned path copy. No substantial independent classifier remains. |
| `resourcemodel/conditional.go` | The external contract is one exact API/result combination. Forwarding requires a complete straight-line instruction census, at most one nested call, exact returned-call identity, an access-path walk and a caller relationship proof. The instruction switch collects that census; translating its few accepted forms would retain the same stateful collection and identity queries in Go. |
| `ssaflow/cfg` | Instruction indexing/order, dominance/reachability, cycle detection, successor filtering and keyed work lists. State-transfer policy is supplied by consumers; this package owns traversal and allowance accounting. |
| `testvariant/analyzer.go` | Prerequisite marker declaration and an analyzer copy with one added prerequisite. |
| `calls/call_constants.go`, `call_conditions.go` | The substantial fixed-outcome recognizer already uses `fixed_outcome.rules`. Remaining code binds argument/capture metadata, validates type/position compatibility, builds bitsets, filters successors and interprets small outcome enums. Bound-condition evaluation unwraps negation, tests the two sides of a nil comparison under budget, and reads the binding map; its native queries and cutoff fences remain the work. |
| `calls/call_effect_uses.go`, `call_field_effects.go`, `call_effects.go` | Direct-use and field-use policy already use rules. Go owns alias-use recursion, closure/call binding, effect-bit accumulation and memo completeness. Builtin effects are a compact name-to-bitset table; final proof construction interprets the unknown bit and budget state. |
| `calls/call_bindings.go`, `callback_storage.go`, `call_goroutines.go`, `call_resolution.go` | Argument/capture iteration, exact result-slot lookup, readonly-use censuses and reaching walks. The small shape checks select inputs to those searches; capture-before-argument order and loaded-cell handling remain with the searches. |
| `calls/call_graph_memo.go`, `call_summaries.go` | Recursion scopes, cache lifetime, dependent-answer invalidation and caller-supplied computation/fallback callbacks. These are query infrastructure, not analyzer proof policy. |
| `calls/call_contracts.go`, `call_symbols.go` | The library registry dispatches explicit contract IDs to exact-symbol families, one documented structural registrar name, or provenance helpers. Most branches are already declaration lists; converting them would require native identity/provenance hooks around the same catalog. Symbol matching itself owns builtin, promoted-method and generic declaration identity. Neither family benefits from a second dispatch layer. |
| `calls/call_uses.go`, `process_entry.go`, `interface_dispatch.go` | Complete body/use inventories, private caller-chain/cycle checks, and reaching-value/method-set resolution. Availability and identity remain attached to those bounded censuses rather than a separate rule decision. |
| `calls/synchronization` | Operation recognition and mutex-effect admission already use rules. Native code declares exact symbols, extracts receivers/counts, validates integer constants and renders the event enum. |

## Command and test infrastructure review

All 18 authored Go files in `internal/cli` and all three authored files in
`testsupport/analyzertest` and `testsupport/docexamples` were read. The bounded
graph query returned exactly those 21 implementation paths. Exact-path metadata
was clean at generation `2026-10-21T23:33:45Z`. The CLI scope also recorded a
partial parse of a generic-method toolchain fixture at line 7; that source was
read directly and is an illustrative input, not an implementation candidate.

| Family | Disposition |
| --- | --- |
| `cli.go`, `selection*.go` | Flag parsing, request validation, selection-map construction, command routing and vet-tool handshake. Base selection has three actual construction strategies: all, selected groups, or defaults; the other switch arms establish an empty base. Precedence then comes from applying group/name/explicit/check-owner updates in order. A rule selector would preserve the same collection loops and map mutations in native hooks. The final per-check admission expression is small. |
| `delegated_run.go`, `diagnostic_json.go` | Process execution, JSON document merging, normalized diagnostic construction, deduplication and sorting. Exit precedence is the compact error/findings/success case distinction. These are I/O and wire-format operations rather than analyzer evidence policy. |
| `diagnostic_render.go`, `documentation.go`, `writer.go` | Terminal and catalog rendering, source snippet layout, flag/version/help output and best-effort writes. Range validation and presentation defaults stay beside the formatting they protect. |
| `dump*.go`, `facts.go`, `heap.go`, `ssa.go` | Existing view table, package/analysis loading, graph/fact enumeration, deterministic ordering and rendering. Trace grouping/folding presents emitted records; it does not reconstruct proof policy. No second analyzer decision belongs in a dump view. |
| `testsupport/analyzertest` | Installs diagnostic-contract assertions and the common analysistest harness. Independent assertions intentionally accumulate test errors before forwarding an unsuppressed report. |
| `testsupport/docexamples` | Fixture-region scanning, package loading/checker execution, position translation, diagnostic attachment and timing. The marker scanner tracks one open region with two marker actions; the surrounding work is text/offset collection and error reporting. Its small kind table and metric setters need no rule adapter. |

## Result and lifecycle fact review

All six result-fact files and the 23 lifecycle-fact files present at the start
of this pass were read, including the three structural-recognition binding
files. Complete path queries and exact-source coverage checks used generations
`2026-10-21T23:33:45Z` and `2026-10-21T23:37:20Z`; authored paths had clean
metadata. The eight recorded gaps were generated outputs. All eight source
policies were inspected directly, including the renamed handoff policy.

This review found two missed fits now integrated, and an evidence-composition
family still awaiting migration. The lifecycle scope is therefore not closed.

| Family | Finding |
| --- | --- |
| `resultfacts` | Static leaves, assumption fallback and paired-return decisions already use rules. Remaining code owns result/parameter enumeration, reaching/storage queries, every-return agreement, summary availability/publication, schema bounds and rendering. Small outcome projections and literal checks stay in Go. |
| Direct and retained results | The retained-result walk duplicated the fresh-result walk's structural switch. Both now consume `lifecycleforms.Handoff`. The existing direct cleanup and imported wrapper checks, exact result-slot agreement and depth bound remain separate. Focused tests distinguish cleanup calls from wrapper calls, direct returns from wrapper returns, and accepted aliases from opaque casts/loads/captures. |
| Returned-view policy | `returned_view.rules` now owns the ordered unchanged-return, possible-release and cleanup-capability decisions. Its final stored-field query retains the method-completeness fence and field census. Overlapping evidence tests pin unchanged/released precedence, type-only acceptance despite unknown methods, and field-based acceptance only with complete method knowledge. |
| Fact vocabulary and publication | Masks, compact claims, package markers, immutable publication and signature attachment. The separate `Reason` file slot was reclaimed by moving unchanged proof carriers beside evidence requests and their codes/rendering beside fact descriptions; the generated view policy keeps the package within its file limit. |
| Field/cleanup, capture and discharge inference | Constructor/method and argument/capture censuses, exact stored paths, local/imported lookup, conditional-case enumeration and shared completion/flow queries. Resource/cleanup vocabularies remain tables. The small Boolean-use admission set in conditional export is a type-set predicate, not an independent decision pipeline. |
| Retention/escape and response leaves | Existing generated policies own instruction admission, mode gates, escape aliases and exact response Body recognition. Native code applies plans through storage/reaching/use searches, memoization and effect collection. |
| Lifecycle evidence composition | **Pending strong fit:** `selectedMaskProof` orders direct discharge, ordinary arguments, immutable captures, strict projections, argument cases and possible-alias uncertainty. `localProof` and `importedProof` also contain source-precedence policy. They require a cohesive native binding design that preserves partial-proof payloads, cutoff precedence, provenance and reached-query order. They remain an explicit migration item. |

## Fixed completion checklist

The inventory is frozen at `4abb415f` plus this batch on 2026-10-22: 363
current authored production Go files, 129 reviewed and 234 awaiting review.
Reviewed source is tracked by content fingerprint. Review each remaining scope
once; reopen only the portion affected by a subsequent change. A review may
close with no migration when the existing Go is clearer. Target count is not a
completion criterion, and marginal conversions do not extend this checklist.

| Remaining source scope | Authored files | Status |
| --- | ---: | --- |
| `analyzers/concurrency/concurrentcapture` | 5 | Pending review |
| `analyzers/concurrency/goroutineownership` | 23 | Pending review |
| `analyzers/concurrency/lockorder` | 24 | Pending review |
| `analyzers/concurrency/producerlifecycle` | 4 | Pending review |
| `analyzers/resources/cancellationownership` | 11 | Pending review |
| `analyzers/resources/deferinloop` | 6 | Pending review |
| `analyzers/resources/processownership` | 13 | Pending review |
| `analyzers/resources/resourcelifetime` | 28 | Pending review |
| `passes/concurrencyfacts` | 25 | Pending review |
| `engine/heapmodel` | 22 | Pending review |
| `engine/lifecycle` | 33 | Pending review |
| `engine/ssaflow` root, `path` and `valueforms` | 40 | Pending review |

Analyzer and pass paths above are relative to `internal/analysis`; engine paths
are relative to `internal`. Counts include native bindings beside existing
rules, which need review rather than automatic conversion.

The remaining completion steps are finite:

- Close each of the 12 source scopes above with a disposition per policy family.
- Migrate the confirmed lifecycle evidence-composition family. Record any other
  confirmed fit under its existing scope and close it with that scope.
- Review generation/build/CI configuration and non-Go tooling for integration
  gaps, without treating language translation itself as a goal.
- Run final integration validation after the remaining cohesive migrations,
  record the outcome and close the audit. Unchanged reviewed scopes and
  compiler contracts reuse their existing evidence.

## Remaining audit work

The internal scope still needs a complete source review with per-family
dispositions. Earlier migration work and the integration guide establish many
positive integrations, but do not establish that every remaining strong fit
has been found. In particular:

- Review smaller recognizers and remaining mixed policy/traversal functions.
- Migrate the confirmed lifecycle evidence-composition family above before
  closing the lifecycle-fact scope.
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
