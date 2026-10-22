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

This review found two structural fits and one evidence-composition family.
All three are now integrated, and the lifecycle-fact source scope is closed.

| Family | Finding |
| --- | --- |
| `resultfacts` | Static leaves, assumption fallback and paired-return decisions already use rules. Remaining code owns result/parameter enumeration, reaching/storage queries, every-return agreement, summary availability/publication, schema bounds and rendering. Small outcome projections and literal checks stay in Go. |
| Direct and retained results | The retained-result walk duplicated the fresh-result walk's structural switch. Both now consume `lifecycleforms.Handoff`. The existing direct cleanup and imported wrapper checks, exact result-slot agreement and depth bound remain separate. Focused tests distinguish cleanup calls from wrapper calls, direct returns from wrapper returns, and accepted aliases from opaque casts/loads/captures. |
| Returned-view policy | `returned_view.rules` now owns the ordered unchanged-return, possible-release and cleanup-capability decisions. Its final stored-field query retains the method-completeness fence and field census. Overlapping evidence tests pin unchanged/released precedence, type-only acceptance despite unknown methods, and field-based acceptance only with complete method knowledge. |
| Fact vocabulary and publication | Masks, compact claims, package markers, immutable publication and signature attachment. The separate `Reason` file slot was reclaimed by moving unchanged proof carriers beside evidence requests and their codes/rendering beside fact descriptions; the generated view policy keeps the package within its file limit. |
| Field/cleanup, capture and discharge inference | Constructor/method and argument/capture censuses, exact stored paths, local/imported lookup, conditional-case enumeration and shared completion/flow queries. Resource/cleanup vocabularies remain tables. The small Boolean-use admission set in conditional export is a type-set predicate, not an independent decision pipeline. |
| Retention/escape and response leaves | Existing generated policies own instruction admission, mode gates, escape aliases and exact response Body recognition. Native code applies plans through storage/reaching/use searches, memoization and effect collection. |
| Lifecycle evidence composition | `evidence.rules` owns local/imported precedence and `selected_mask.rules` owns mask selection. The existing facades call generated policies; native adapters bind optional requests, perform the original searches and construct proofs. Whole partial proofs retain method, provenance and cutoff reasons. Positive local evidence skips imported lookup; an abandoned completion survives a failed transfer, while an independent positive transfer can settle it. Fresh direct/retained result searches share `result_handoffs.go`, and invocation/complete summary inference share `summarize.go`, leaving room for both generated outputs within the package's file limit. |

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
| Abandoned-send decision | `abandoned.rules` now owns reachable-return, producer-count availability, receive-count uncertainty/zero and final excess precedence. The count and return searches remain native. Unknown count proofs retain their complete payload; later receive queries are skipped. |
| Helper receive evidence | `helper_receive.rules` now owns launch/builtin exclusions, summary availability/cutoff, nonreceiving-worker evidence and opaque channel consumption. Native capture inventories retain short-circuit queries; exact counts and the opaque-helper uncertainty reason survive, including its false payload. |
| Producer/receiver census | Worker discovery, parameter/capture mapping, complete protocol filtering, source attribution, dominance chains, per-worker send order and shared-budget exact receive counts. The collection loops and storage/call-effect searches remain native. |
| Rendering and reporting | Stable reason table, proof carriers, tracing and position deduplication need no additional rule layer. |

Both confirmed sequences now have generated production consumers and their
former Go decision paths are removed. Focused tests retain cold/warm budget
boundaries and pin early origin/builtin exclusions, caller-return/count
precedence and the false uncertainty payload of an unrelated opaque helper.
Existing trace tests pin the original reason/outcome pairs. This scope is closed.

## Defer and process source review

All six authored `deferinloop` files and all 13 authored `processownership`
files were read, alongside their two and five rule files respectively. The
bounded graph query returned all 68 indexed paths across the two scopes,
including tests/fixtures. Exact-path coverage for the 19 implementation paths
was clean at generation `2026-10-22T00:04:48Z`. The six scope gaps were ignored
generated outputs; all source policies and their shared schemas were read
directly. Generated calls into defer classification are absent from the graph's
caller results, so source rules establish those consumers.

| Family | Disposition |
| --- | --- |
| Defer classification and acquisition | `classify.rules` owns transfer/opaque/cleanup/argument precedence; `acquisition.rules` owns typed-resource, response-body and imported-owned-result selection. Keep the existing rules. The wrapper that optionally classifies the owning response is compact composition. |
| Defer transfer evidence | `transfer.rules` now orders six structural handoff queries and imported argument retention as lazy alternatives with the original Boolean result. An exact global transfer test supplies no imported evidence context, pinning the early exit. |
| Defer argument-use evidence | `argument.rules` now selects wrapper uncertainty and exact release per argument. The native census retains alias/containment evidence and its trace event before label selection, accumulates exact-argument use and delays unavailable-callee uncertainty until all arguments have been examined. |
| Defer obligation and flow | Exact standard lock-pair table, acquisition/store censuses, storage identity, live-backedge traversal, iterator exhaustion, monotone status and witnesses. These remain with the flow/search algorithms. Final unknown-versus-live precedence is compact; no general rule engine should replace the keyed walk. |
| Process instruction action and final return outcome | Existing action entry/settlement/remainder rules and `decision.rules` own substantial precedence. Native code binds lifecycle requests, allocates query allowances, memoizes candidate actions and projects the selected decision to tracing/reporting. |
| Process pre-Start decision | `prestart.rules` now orders completed-prefix availability/cutoff, helper/caller/aggregate ownership, pre-registration and successful-return evidence. The native query methods allocate the original child allowance only when reached; the prefix census still precedes suppression queries. |
| Process return admission | `return_admission.rules` now selects failed Start, returned command owner, returned process handle and merged owner in order. The former closure table is removed. The flow callback emits the selected reason once and retains witness/feasibility collection. |
| Possible Wait participation | `handoff.rules` now sequences exact Wait/merged identity, visible spawned-body reachability/completion and opaque callback capture. Native queries retain cutoff proof payloads, early visible-body exits and callback-value-before-argument capture order. Uncertain completion keeps Method and Provenance. |
| Process local searches | Deferred capture/argument censuses, command-use/handle provenance, registered-owner discovery, domination and returned-handle containment remain native. Their small shape switches feed bounded searches rather than form independent rule pipelines. The immediate Process guard validates an indexed four-instruction block plus identity/edge evidence; a rule adapter would still need that native block decoding. |
| Symbol contracts and presentation | Exact library-symbol lists, pipe/Wait type contracts, reason tables, names, report construction and trace projections remain direct Go. No second catalog or source rendering layer is needed. |

These two source reviews are complete. The process scope is closed after its
three migrations; the defer scope is also closed after its transfer and
argument-use migrations. The recorded traversal and presentation families are
not reopened without an affecting change.

## Cancellation source review

All 11 authored `cancellationownership` files and all eight rule/schema files
were read. The graph returned all 44 indexed paths, including fixtures/tests;
exact-path metadata was clean at generation `2026-10-22T00:15:39Z`. Seven scope
gaps were ignored generated classifiers, covered by direct rule/schema reads.
Same-named method edges such as `Parent` and `Common` resolve unreliably; the
exact source establishes parent-frame identity and reached-query order.

| Family | Disposition |
| --- | --- |
| Instruction/call/return classification | Seven generated policies already own ordinary, direct, argument, deferred, call-contract and return-label precedence. Preserve those rules and native request binding. |
| Parent context origin | `parent.rules` now recognizes exact parent result zero in the same frame and binds the constructor's cancel result. `parentCancellationClassifier` retains the standard-child gate and acquisition-time storage snapshot before recognition. The ordinary context-origin recognizer has a different contract; wrappers, merges, other result slots and foreign frames remain opaque. |
| Directly deferred literal | `literal.rules` now sequences closure capture availability, result-guard membership, completion and cutoff. The native capture/guard censuses retain one child allowance, allocated only after closure recognition. A nonliteral defer test supplies no classifier, pinning the absence of a budget query. |
| Constructor-owner evidence | Cached field/store/closure-use inventories require one once-written cell, initialization before capture, one owner field and a complete visible-use census. Budget cutoff discards all accumulated holds. Small structural branches select what to add to those inventories rather than an independent decision pipeline. |
| Result guards and local uses | Guard discovery/retention, per-return reachability/completion, memoized every-use resolution and exact local wrapper/load folds. These keep native iteration, aggregation and recursive query state. The selected Done edge is a narrow query over existing select recognition. |
| Flow and action orchestration | Classifier construction, parent/return label combination, candidate-local memoization, one obligation flow and compact outcome/lattice projection. Retain these with the cache and flow; the substantial classifiers already use rules. |
| Contracts and presentation | Exact standard constructor table, callback/cleanup/Done symbol contracts, small instruction-to-reason and outcome mappings, naming, tracing and reporting. These are compact declarations or rendering rather than substantial standalone recognizers. |

The cancellation source review and both confirmed policy migrations are
complete. Other families are not reopened without an affecting change.

## Goroutine ownership source review

All 23 authored Go files are reviewed: the seven helper files at generation
`2026-10-22T00:33:39Z` and the 16 root files at `2026-10-22T00:38:13Z`.
Exact-path metadata was clean; the 14 scope gaps were ignored generated
outputs. All rule/schema sources were read directly, including the four new
policies. Graph callers omit function-valued callbacks, so exact source also
establishes edge-flow consumers and the three notification consumers.

| Family | Disposition |
| --- | --- |
| Proof and instruction precedence | Existing `proofpolicy`, entry, call-prefix/contract and classify rules own substantial admission, lifecycle/fallback and label choices. Native code keeps action caches, outcome/lattice projection, trace effects and concrete search dispatch. |
| Caller lifetime | `caller_lifetime.rules` replaces the ordered query-closure table. One native allowance is bound once; each query records post-search availability before a witness can name a lifetime reason. Unknown remains a terminal answer. |
| Edge actions | `edge.rules` orders exact receive, counted drain, possible receive and retained-context evidence. Drains forward both exact and unknown answers without later queries. Native hooks retain identity/count searches and the original reason-recording point. |
| Aggregate projection | `containment/aggregate.rules` replaces the six-form source switch. The native walk retains explicit wrapper transparency and per-step charges. Index operands, receives, slices and tuple extracts are not silently followed; a copied struct parameter retains its own root. |
| Completion notifications | `completionsteps/notification.rules` now selects notifications for direct, nested and every-return discovery. Detached notifications are excluded first; only sends request terminal coverage. An ordinary close in a cycle is per-item cleanup; deferred closes still require registration coverage from their caller. |
| Discovery and binding | Channel/group/owner/pipe-peer inventories, exact capture-before-parameter mapping, stable captured cells, nil-actual exclusions, callback resolution and factory-result censuses remain native. Their small leaf gates select inputs to those searches. Constructor discovery applies evidence and observes cutoff while accumulating candidate state. |
| Completion, helper and retained-owner searches | Terminal-work queues, return coverage, bounded helper/receive memoization, caller/context field-write censuses, retention reaching walks, cleanup-factory relation searches and summary-operation enumeration remain native. Call-form/receiver exits stay beside the binding scans that use them; another query-context adapter would retain those same mechanics. |
| Counted drains and guarded joins | Exact sender/cell/closure-use censuses, known loop counts, correlated flag/counter stores and path queries remain with their algorithms. No rule guesses a loop's runtime count. Small selectors and final evidence projections do not need separate adapters. |
| `containment`, `origins`, `receives` | Existing leaf/stored-use/escape and receive rules retain their structural policy. Reaching/referrer walks, sibling address enumeration, bounded origin storage, per-arm charges and caller channel predicates remain native. Their results describe possible ownership unless an exact query proves otherwise. |
| Contracts and presentation | Exact API symbol tables, lifecycle name vocabulary, numeric reason spellings, source ranges, report construction and trace rendering remain Go declarations and projections. |

The source review and all four confirmed migrations are complete. The scope is
closed; unchanged families are not reopened. Ordered-policy tests pin lazy
query order and partial-cutoff precedence; compiled-SSA and trace fixtures pin
exact versus possible ownership, notification registration and root boundaries.

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

The three confirmed integration gaps are corrected:

- `scripts/benchmark-dogfood.sh` generates ignored sources before its own binary
  builds, including when invoked directly instead of through Make.
- `scripts/precision-regression.py` does the same when it builds a replay binary.
  Supplying an existing binary retains the existing no-build path.
- `.pre-commit-hooks.yaml` uses `language: system` and runs the installed CLI.
  The installation guide explicitly requires that binary on `PATH`, including
  in CI, instead of promising an isolated Go source build.

The CI policy tests exercise successful generation-before-build ordering and
failure propagation for both script entry points with isolated command stubs.
No corpus checkout or instrumented Go build is needed for those tests. The
source-install limitation itself remains documented in `docs/installation.md`;
release archives are generated before compilation. This integration review is
closed; no cross-language migration is planned.

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
| `analyzers/concurrency/goroutineownership` | 23 | Reviewed; lifetime, edge, aggregate and notification policies migrated |
| `analyzers/concurrency/lockorder` | 24 | Pending review |
| `analyzers/concurrency/producerlifecycle` | 4 | Reviewed; evidence sequencing migrated |
| `analyzers/resources/cancellationownership` | 11 | Reviewed; parent-origin and deferred-literal policies migrated |
| `analyzers/resources/deferinloop` | 6 | Reviewed; transfer/argument policies migrated |
| `analyzers/resources/processownership` | 13 | Reviewed; three decision families migrated |
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
policy. Current source totals are 360 authored files: 228 reviewed and 132 still
awaiting review across 5 source scopes. All previously confirmed migrations
are integrated; remaining candidates must come from those unreviewed scopes.

The remaining completion steps are finite:

- Close each remaining source scope above with a disposition per policy family.
- Migrate any confirmed fit under its existing scope and close it with that scope.
- Preserve the completed source-build/pre-commit integration corrections above
  during the remaining migrations; reuse their current validation.
- Run final integration validation after the remaining cohesive migrations,
  record the outcome and close the audit. Unchanged reviewed scopes and
  compiler contracts reuse their existing evidence.

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
- Reuse the completed non-Go integration review and its validation unless the
  generation/build/CI inputs change.
- After remaining migrations, verify consumers and removed decision paths,
  generation, relevant behavior and the final repository gate. Reuse unchanged
  validation rather than repeatedly running a broad gate during discovery.

The audit should close these scopes before claiming that the goal is complete.
