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

## Shared value and path review

The 40 remaining authored files in `engine/ssaflow` were read in full, covering
the root and `path`, `valueforms`, `selection` and `ssaflowtest`. The bounded graph
query returned all 106 indexed files under `ssaflow`, including the already
reviewed call/CFG scopes and tests. Exact-path coverage for all 40 authored
candidates was clean at generation `2026-10-21T23:51:40Z`. The seven child-scope
gaps were ignored generated outputs; their source rules and shared schemas were
read directly. Method-name resolution can confuse unrelated `unknown` symbols;
exact source, rather than that call edge, establishes the identity fallback.

| Family | Disposition |
| --- | --- |
| `value_identity.go` | `identity.rules` now sequences structural identity, left-path lookup, right-path lookup and path comparison. Each cutoff remains unknown with its original reason and provenance. Direct identity still precedes nameable-path requirements. Missing inputs and failed left paths do not start later searches. The native equality and projection searches retain their recursion and charges. |
| `access_paths.go`, `value_field_path.go` | Bounded path construction, exact root checks, constant-index encoding and load snapshot selection. The three projection branches update or extend the path while recursing; a separate descriptor would repeat this dispatch without removing traversal policy. Embedded paths append checked field indexes to a fixed buffer. |
| `value_reaching.go`, `value_phi.go`, `value_derivation.go`, `value_matching.go` | Generic any/every/agreement folds, cycle handling, sibling-state cloning, use/operand censuses and directional structural matching. Field/index/load branches drive different recursive searches, rather than choose a standalone answer. Keep origin-before-expansion and exact-address-store semantics with these algorithms. |
| `value_forms.go`, `valueforms`, `value_ownership.go` | Caller-selected wrapper and ownership-source recognition already use rules. Remaining forwarding and terminal-value tests are compact type sets; allocation stores and aggregate membership retain native reaching/use walks. |
| `channel_aliases.go`, former `channel_types.go`, `named_results.go`, `written_once_cell.go` | Complete alias, store, capture and return-slot censuses with initialization/dominance checks. Small type switches select which census to extend; uniqueness, capture recursion and exact observation boundaries remain with the walk. The one channel-type predicate now lives with channel alias mechanics. |
| `natural_loops.go`, `slice_elements.go`, `path/counted_loop.go` | Backedge/region collection, induction-edge agreement, invariant bounds and exact variadic-write inventories. Small header/increment/length shapes are local prerequisites of these algorithms; a rule adapter would leave their graph/edge validation and mutable collections intact. |
| `selection`, `path/flow_select.go`, `path/flow_branch_literals.go` | Select-dispatch and literal-comparison recognizers already use rules. Incoming-phi lookup, all-return literal agreement, helper-local caps and exact receive-edge checks stay native. |
| `path/flow_guards.go` and guard support | Substantial loaded/stable/address recognition already uses rules. The remaining code owns memoized rendering, bounded guard sets, contradiction comparison, invalidation, byte-compatible keys and IDs. The small contradiction lattice and store/call dispatch do not justify another plan/apply layer around those operations. |
| `path/flow_assumptions.go`, `flow_successors.go` | Successor narrowing composes literal, fixed-value and exact nonnil/type evidence, retaining cutoff after each query. The code directly filters edge sets and carries predecessor/type context. Its short structural prerequisites and three-stage orchestration remain clearer beside that edge operation. |
| Other `path` flow files | Work-list traversal carries classifier actions, guards, coverage and witnesses. Termination checks stop only their own paths and scan dominating deferred registrations. Entry modes, outcome lattices and Boolean facades are compact dispatch; classifiers remain the authoritative domain policy. |
| `source_metadata.go`, `value_instructions.go`, `return_values.go`, `storage_integer.go`, `doc.go`, `ssaflowtest` | Source/closure/instruction enumeration, metadata accessors, literal extraction, all-result validation, package documentation and SSA fixture construction. These are collections, narrow mechanics or test infrastructure without a substantial ordered recognizer to extract. |

The identity tests retain full proof comparison, corresponding/different/dynamic
paths, shared allowance and final-comparison cutoff. New cases pin missing
operands, an already exhausted budget and left-before-right query behavior.
The shared-flow scope is closed after this migration; it is not reopened merely
to increase the target count.

## Repeated-capture analyzer review

All five authored `concurrentcapture` files and its four rule files were read.
The graph returned all 15 indexed paths, including tests and fixtures; exact
implementation-path coverage was clean at generation `2026-10-21T23:54:49Z`.
The four scope gaps were ignored generated outputs, whose rules were inspected
directly. This scope is closed without further migration.

| Family | Disposition |
| --- | --- |
| Mutation decision and synchronization region | `mutation.rules` already owns held-lock/fallback/worker/channel guard precedence; `lock_transition.rules` owns acquire/release/unknown selection. Native code applies selected operations to the held map and keeps uncertainty sticky. |
| Mutation roots and semaphore statements | `root.rules` and `channel_statement.rules` already own root peeling and direct send/receive recognition. Object resolution, iteration-local version checks, scoped AST searches and the pending-send census remain native. |
| Worker-prefix evidence | The complete ordered prefix scan requires brokered call summaries and declines conditional/unknown ordering. Target-instruction uniqueness, call-operation collection and compact final held/unknown projection stay with that scan. A descriptor would preserve the same loop and queries in hooks. |
| Analyzer orchestration and presentation | Loop/closure discovery, binding of varying parameters, candidate-local guard requests, report deduplication, stable reason rendering and diagnostic construction. Small AST dispatch and enum-to-string mapping do not justify additional adapters. |

## Producer lifecycle review

All four authored `producerlifecycle` files and `receive.rules` were read.
The graph returned all 22 indexed paths, including tests/fixtures. Exact-source
coverage was clean at generation `2026-10-21T23:54:49Z`; the one scope gap was
ignored generated receive recognition. Its source policy calls `helperReceives`
even though the graph has no incoming call edge for that generated consumer.

| Family | Disposition |
| --- | --- |
| Instruction receive recognition | Already generated. Direct receives, select-state census and helper results preserve count, uncertainty and reason. |
| Abandoned-send decision | **Pending strong fit:** `abandonedProducerSend` orders reachable-return evidence, producer-count availability, receive-count uncertainty/zero and the final excess comparison. The count and return searches remain native; a generated policy can own the evidence precedence and proof selection. |
| Helper receive evidence | **Pending strong fit:** `helperReceives` orders launch/builtin exclusions, summary availability/cutoff, nonreceiving-worker evidence and opaque channel consumption. Preserve short-circuit capture queries, exact counts, reached-query order and the existing uncertainty reason, including its false payload. |
| Producer/receiver census | Worker discovery, parameter/capture mapping, complete protocol filtering, source attribution, dominance chains, per-worker send order and shared-budget exact receive counts. The collection loops and storage/call-effect searches remain native. |
| Rendering and reporting | Stable reason table, proof carriers, tracing and position deduplication need no additional rule layer. |

The source review is complete, but this scope remains open until the two
confirmed evidence sequences are migrated and their production consumers use
the generated paths.

## Non-Go integration inventory

The tracked non-Go code inventory outside fixtures/skills contains eight Python
files, seven shell scripts, one jq program, five MJS files, seven TypeScript
files, nine Astro components and two codec benchmark protobuf schemas. These
own audit automation, build/process/resource management, website rendering and
codec inputs. Hyrule emits typed Go rather than those languages; translating
these tools into Go solely to use rules would add a maintenance boundary.
The relevant integration review is therefore generation before native builds.

The on-disk compiler symlink and ignored workspace select the local Lockdoc
module. `scripts/hyrule.sh`, `go:generate`, Make's generation prerequisite,
`prepare-hyrule`, CI, minimum-toolchain testing, release archive building and
site package scripts were inspected. Hosted jobs generate before switching to
standalone module dependencies; the release build and CI's direct binary build
follow that preparation. The pin still matches the local compiler source.

Three existing integration gaps are confirmed and remain in this checklist:

- `scripts/benchmark-dogfood.sh` directly builds GoHawk without first generating
  its ignored sources; the `benchmark` Make target does not supply generation.
- `scripts/precision-regression.py` similarly builds its own binary without
  generation when no existing binary is supplied. Its Make target does not
  supply generation either.
- `.pre-commit-hooks.yaml` still asks pre-commit's Go environment to compile the
  repository, which cannot contain ignored generated Go on a fresh checkout.
  The installation guide still describes that isolated-build contract.

The first two entry points need generation before their own source builds.
The pre-commit hook needs an explicit supported installation contract. These
are integration corrections within the existing final-check scope, not new
analyzer conversions. The source-install limitation itself is already documented
in `docs/installation.md`; release archives are generated before compilation.

## Fixed completion checklist

The inventory was frozen at `4abb415f` plus the lifecycle handoff/view batch
on 2026-10-22: 363 authored production Go files, 129 reviewed and 234 awaiting
review at that point.
Reviewed source is tracked by content fingerprint. Review each remaining scope
once; reopen only the portion affected by a subsequent change. A review may
close with no migration when the existing Go is clearer. Target count is not a
completion criterion, and marginal conversions do not extend this checklist.

| Remaining source scope | Authored files | Status |
| --- | ---: | --- |
| `analyzers/concurrency/concurrentcapture` | 5 | Reviewed; existing rules cover substantial policies |
| `analyzers/concurrency/goroutineownership` | 23 | Pending review |
| `analyzers/concurrency/lockorder` | 24 | Pending review |
| `analyzers/concurrency/producerlifecycle` | 4 | Reviewed; evidence sequencing pending migration |
| `analyzers/resources/cancellationownership` | 11 | Pending review |
| `analyzers/resources/deferinloop` | 6 | Pending review |
| `analyzers/resources/processownership` | 13 | Pending review |
| `analyzers/resources/resourcelifetime` | 28 | Pending review |
| `passes/concurrencyfacts` | 25 | Pending review |
| `engine/heapmodel` | 22 | Pending review |
| `engine/lifecycle` | 33 | Pending review |
| `engine/ssaflow` root, `path`, `valueforms`, `selection` and `ssaflowtest` | 40 | Reviewed; identity policy migrated |

Analyzer and pass paths above are relative to `internal/analysis`; engine paths
are relative to `internal`. Counts include native bindings beside existing
rules, which need review rather than automatic conversion.

The shared-flow review closes 40 frozen paths; the small channel-type file was
merged into channel alias mechanics to leave room for the generated identity
policy. Current source totals are 362 authored files: 177 reviewed and 185 still
awaiting review across 9 source scopes. Lifecycle evidence composition and
producer evidence sequencing remain open migration items separately.

The remaining completion steps are finite:

- Close each remaining source scope above with a disposition per policy family.
- Migrate the confirmed lifecycle evidence-composition and producer evidence
  sequencing families. Record any other confirmed fit under its existing scope and close it with that scope.
- Correct the three confirmed source-build/pre-commit integration gaps above.
  The non-Go inventory review is complete; no cross-language migration is planned.
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
- Close the three confirmed non-Go integration gaps recorded above; reuse the
  inspected generation/build/CI paths unless their inputs change.
- After remaining migrations, verify consumers and removed decision paths,
  generation, relevant behavior and the final repository gate. Reuse unchanged
  validation rather than repeatedly running a broad gate during discovery.

The audit should close these scopes before claiming that the goal is complete.
