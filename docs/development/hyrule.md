# Hyrule integration

Hyrule generates structural recognizers and ordered decision functions from
rules. Runtime builds compile the generated Go; analysis does not invoke the
compiler or interpret rules. Generated Go and the ownership manifest are
committed, so a fresh checkout compiles without the generator. CI checks that
committed output matches its rules before regenerating anything; commit
regenerated output together with the rule change that produced it.

## Local development workspace

`tools/hyrule` links to `../../lockdoc/tools/hyrule`. Create the ignored local
workspace with `go work init . ../lockdoc`; it selects this repository and the
adjacent `../lockdoc` module. This intentionally follows
the generator on disk rather than pinning or copying it. The workspace requires
Go 1.27; the standalone GoHawk module retains its own Go floor and dependencies.
`make hyrule-generate` builds the workspace-selected generator and updates root
`hyrule.yaml` targets. `make hyrule-generated-check` verifies freshly generated outputs;
`make hyrule-tool-test` runs the linked compiler's contract tests.

The expected checkout layout is a common parent containing `gohawk` and
`lockdoc`. `make build` and other compilation targets generate from the adjacent checkout
first. After generation, `GOWORK=off` builds against GoHawk's own dependencies.
Each CI build job generates with a pinned Lockdoc revision, then tests the
standalone GoHawk module, including its supported Go floor. The Hyrule manifest
records generated-file ownership and hashes. Edit rules or bindings and
regenerate, never patch generated Go.

Each package keeps its rule sources in a `rules/` folder beside the Go file
holding its bindings. Generated `*_gen.go` output stays in the package directory
itself, since it calls unexported host functions. It is marked as generated
for GitHub and does not count toward the directory file limit.

## Function frontend

Function headers attach the API to its rules:

```text
total fn classify(request: Request) -> Decision
  fallback Unknown()
  match request
    @classify_L4_F4ac38f99
    Request { ready: true } ->
      Accepted()
```

`partial fn` returns native outputs plus a success Boolean. Plain `fn`
retains the original one-input/optional-host API. Most GoHawk targets declare
their API in Go: `total fn rewrite` omits duplicate annotations, then
`match inputs` uses the existing native parameter bundle patterns.
Equation fragments and schema files remain shareable between targets.
Arm captures and local values infer their types. `@Label` names remain stable,
and `when` qualifiers preserve the original guard order.

The handwritten parser is specified by
[grammar.ebnf](../../tools/hyrule/grammar.ebnf) and
[LAYOUT.md](../../tools/hyrule/LAYOUT.md). The
[Hyrule reference](../../tools/hyrule/README.md) covers all language forms.
`?` marks host matchers, `!` marks ordered queries, and unmarked aliases bind
pure functions or constants. Concrete interface patterns use Go compatibility;
there is no `implementers` declaration or claim that interfaces are closed.
Standalone `signature` declarations are retired. Named fields use braces:
`Request { header, count } as request` destructures by field name and captures
the whole record. `{ count }` means `{ count: count }` in patterns, construction
and updates. Updates use `request with { count: next }`; nested records can
use their own `as` captures. Prefix `name@pattern` captures and parenthesized
named fields/updates are no longer accepted.

## Query sequencing and schema maintenance

The HTTP acquisition policies use `do` blocks for dependent, stateful queries.
The success-branch policy combines `do` with ordered alternatives: explicit
`decline` tries ordinary flow after an unavailable assertion strategy, without
repeating its queries or refunding its allowance.
Conditional steps select a complete outcome and explicitly return on every
other outcome. An unconditional `value <- operation ...` binds the
complete result without inventing a success tag; reached calls still execute
once in order, and declared union results retain validation. Boolean steps may write `false <- operation ...` followed by `else result` (or `true`)
to handle the complementary literal without a one-arm block. Tagged outcomes
retain explicit named arms. Reached queries retain their original order and budget charges; the
public Go entry points still discard evidence if the child or shared allowance
is exhausted. CI pins the compiler revision supporting this syntax.

Concurrency summary snapshots use Boiler's field-complete snapshot operations
in `boiler.yaml`. They preserve the same four copy functions and nested hooks:
branch-mutable lists are detached, while immutable SSA references, path summaries
and condition evidence retain their existing sharing. Each representation field
has an explicit policy. Budgeted copying and fixpoint equality remain separate.
`make boiler-generate` runs before Hyrule; `make boiler-generated-check` checks
the committed snapshot output. The retired `derive ... freeze` rules are removed.

## Migration scope

A strong fit is a substantial ordered structural recognizer, a decision over
explicit evidence, or a finite state-transition policy. Traversal, alias search,
fixed-point scheduling, budgets, caches, and effect application remain in Go.
A trivial enum switch or an already-declarative lookup table is not a migration
requirement merely because a rule could express it.

The [completion audit](hyrule-audit.md) records the per-family disposition of
every reviewed scope and the validation that closed the migration.

The repository-wide migration covers these families. Integration is complete
only when the consuming production path uses the rules and the old decision
path is removed; generator support and examples alone do not qualify.

| Family | Current integration |
| --- | --- |
| Select-index comparison recognition | Generated shared recognition used by summary dispatch and selected-receive edge evidence |
| Standard synchronization primitives | Generated shared recognition used by concurrency call summaries and heap mutex effects; count acceptance remains summary policy |
| Concurrency inert-data and passive-instruction admission | Generated admission over inert-data contracts and budgeted field-path evidence; collection stays in Go |
| Concurrency snapshot maintenance | Generated complete-record copies for summaries, workers and select choices; nested list hooks retain the original copying depth |
| Concurrency context/callback and resource recognition | Generated context calls/origins, callback inputs/calls, resource references and identity-root forms; storage resolution, callback binding and summary construction stay in Go |
| SSA wrapper, projection, and structural source families | Generated transparent wrappers and ownership-source recognition; other projection/source families remain |
| Corresponding-path identity evidence | Generated structural/path/cutoff precedence over the native equality and access-path searches |
| Call-use and heap-operation classification | Generated direct call-use effects, field-use admission, heap value definitions and static heap leaves; memory effects, recursive searches and state application stay in Go |
| Lifecycle completion/transfer recognition and outcome selection | Generated launch, exact/retained callback, binding/enclosing steps, context admission, conditional tests and unproven-completion decisions, plus capture/argument/deferred/receiver mapping, path translation/agreement, local completion strategy and conditional-return plans; recursive coverage, storage queries, invocation environments and the existing transfer table stay in Go |
| Goroutine ownership instruction/call labels | Generated arm-entry, receive, publication, containment, worker-completion instruction plans and helper-escape cases plus ordered call contracts; reaching, callee binding, caches and searches remain in Go |
| Cancellation ownership instruction/call labels | Generated instruction, direct/deferred, call-contract, exact-argument and return decisions; owner discovery, searches, cache and flow stay in Go |
| Process ownership decision selection | Generated pre-Start, return-admission, possible-handoff, post-Start outcome and instruction action policies; handle scans, caches, request binding and flow remain in Go |
| Resource lifetime classification and acquisition-boundary patterns | Generated instruction and opaque-consumption/storage/call policies, HTTP acquisition/effect/use decisions and acquisition-error policies; optional correlation searches stay in Go; catalog collection stays a table |
| Defer-in-loop resource classification | Generated per-value, transfer, argument and acquisition decisions; call discovery and loop traversal remain in Go |
| Lock operation, write-target, and held-state transition families | Generated write destinations, owner-address steps, dynamic-index leaves, identity/loop-origin cases, budgeted copy sequencing and mutex-transfer sequencing; storage/reaching searches, collection mutation and cycle queries stay in Go |
| Concurrent capture syntax and guard decisions | Generated mutation-root steps, channel statements, guard decisions and exclusive-lock transitions; prefix collection and effect application stay in Go |
| Producer lifecycle evidence | Generated receive instruction, helper fallback and abandoned-send decisions; binding scans, count collection, capture inventories and cycle queries stay in Go |
| Result and lifecycle summary leaf recognition | Generated static result leaves, fixed argument outcomes, assumed leaves, conditional return decisions and branch comparisons, plus response-body leaves, freshness, retention and escape-use classification; summary inference and reaching walks stay in Go |
| Path guards and literal comparison | Generated stable-guard, loaded-condition, address-step and literal-operator decisions; traversal, cycle queries, rendering memos, invalidation and flow application stay in Go |
| Diagnostic admission | Generated ordered withdrawn/unknown/selection/suppression/test-file decisions; trace emission, error accumulation, callback restoration and publishing stay in Go |

## Preserved boundaries

Select recognition returns the exact select, integer case number, equality
polarity, and operand order. Summary dispatch accepts reversed and inequality
forms; selected-receive edge evidence still accepts only its original canonical
forward equality. Default-arm and bounds interpretation remain with each caller.
Index extraction must select result zero; receive-ok and payload extraction do
not establish a selected case. Neither recognizer unwraps values implicitly.

Synchronization recognition preserves exact `syntax.Symbol` identity and the
numeric operation vocabulary. Reader and writer modes remain distinct. A
WaitGroup Add with anything other than the exact constant one remains recognized
with its exact delta (or unknown count), so the summary stops collection with the existing reason rather than falling
through to helper inference. Cancellation, callback, NewCond, inert-call and
helper handling retain their previous precedence around the generated primitive
recognizer. Resource identity is resolved afterward by the existing engine.

Transparent-wrapper rules recognize exactly one step under the caller's form
mask. Conversions remain opt-in, comma-ok assertions require result-zero
extraction, and success flags or unrelated tuples remain opaque. Reaching walks
and Boolean-negation traversal retain their existing Go budgets and termination.

## Native host boundaries

Defer-in-loop classification uses a native four-parameter, two-result signature.
Its rules call the existing transfer and argument-evidence functions through
explicit field forwarding. Boolean queries use `true`/`false` arms, and tuple
outcomes declare their discriminator and payload slots. Unrecognized outcomes
return the declared `Unknown reasonNone` fallback, without retrying another
rule. No request, query-host, Boolean-answer or decision wrapper structs remain.
The three small Go hooks still own call presence and the cleanup contract.

Transparent SSA wrapper extraction uses read-only views of the original SSA
nodes. The views generate checked assertions and field reads; the rules retain
form masks and the distinction between an asserted value and its success flag.
The existing reaching walks and ownership proofs remain authoritative.

Ownership-source rules select one syntactic holder candidate for a field,
element, load, receive, assertion, extraction or range operation. The source
keeps its exact SSA identity. Arithmetic/logical unary operations retain a false
recognition result, and traversal stops there. `ExternallyOwnedValue` and
`ElementOfAggregate` consume this same generated recognition through their
existing reaching walks; the rules neither traverse nor establish ownership.

`ssaflow/rules/identity.rules` owns `ProveIdentityWithin`: structural identity wins
before requiring a nameable access path, then left and right path searches run
in order before step comparison. Missing operands or unavailable paths yield
unknown without imported provenance; exhaustion retains the budget reason.
Native traversal and every search charge remain unchanged. The caller must
already have established equivalent roots. This policy does not replace the
path-only contract of `SameAccessPathWithin`.

Cancellation parent-origin rules operate on the already resolved acquisition
snapshot. Only exact result zero of a recognized cancellation constructor in
the same function supplies its cancel result; wrappers, merges and foreign
frames remain opaque. The standard-child contract and observed storage query
stay in Go. Deferred-literal rules sequence capture, guard membership and
completion using the same child budget, created only after closure recognition.
Native censuses retain loop visits; uncertainty never credits cancellation.

Defer resource-transfer rules preserve the structural contracts before imported
retention. Argument rules consume native alias/containment evidence after its
trace event, then distinguish an exact releasing argument from a wrapper.
The native loop accumulates use across arguments and asks whether the callee
was summarized only after no earlier argument settled or obscured the resource.

## Concurrency admission

`concurrencyfacts/admission/rules/data.rules` owns inert-data instruction admission.
The child package also owns the existing inert value/container, builtin/atomic
and standard resource-type contracts. It cannot call back into the enclosing
pass. `passive.rules` owns the remaining instruction/root decision; its field
query invokes the existing bounded path proof exactly once when reached.
Collector traversal, scalar arithmetic, return-value scans and resource identity
remain in Go. The enclosing pass keeps its existing exported mutex predicate as
a delegate to the same type contract.

The rules preserve empty-container admission, select-projection exclusion,
resource-bearing writes, pointer spills and root-only channel allocations.
Unknown forms retain `ReasonEffectUnknown`. `effects_test.go` keeps adjacent
accepted/rejected summaries and checks complete, exhausted and fresh field-path
allowances, plus root/callee allocation with no search charge.

## Context and callback recognition

The `concurrencyfacts/recognition` package owns four native context/callback
recognizers. Context
calls return their operation kind and exact parent, receiver or cancellation
value. Context origins distinguish a Done call from either constructor result.
These are symbolic operations: the enclosing engine still resolves the parent,
requires Background/TODO, binds cancellation identity and constructs the summary.
Library symbol contracts have one declaration shared by recognition and binding.

Callback inputs are exact parameters or captures holding function values or
non-context interfaces. Callback calls admit only synchronous dynamic calls with
inert arguments; static callees, opaque loads, resource arguments, launches and
defers stay outside that recognizer. The engine retains callback substitution,
concrete method lookup, supplied-body analysis and ordered effect construction.
No inferred binding or search moves into a pure matcher.

`recognition_test.go` checks exact SSA operands, constructor projections, nil
inputs, misleading API names and callback scope on compiled SSA. Existing
`cancellation_test.go` and `callbacks_test.go` retain the parent, identity,
interface, capture, imported-summary and uncertainty controls.

## Cancellation labels

The cancellation classifier uses generated decisions for instruction admission,
direct and deferred use, call contracts, exact callback completion, and returns.
All seven targets share a closed label domain and a recognized/unrecognized
outcome. A recognized `none` label still stops recognition: it cannot fall
through to a later opaque-use rule. Exact cleanup remains available after owner
discovery stops, while unavailable owner evidence still suppresses default loss.

Receiver methods bind directly to explicit classifier arguments. Whole view
captures retain the concrete asserted SSA type, so deferred-literal queries use
the original `*ssa.Defer` without a conversion adapter. Completion's request and
spent child allowance travel together into summary fallback. The existing action
cache remains the sole cache and trace boundary, and parent/return combination
continues to feed the same obligation-flow query.

First-answer policies use `do` pattern binds or exhaustive `case` decisions
with explicit `decline` arms, joined by `<|>` and ending in a pure fallback expression. Each query is explicit and lazy; unselected valid outcomes
continue inside the committed rule. Answers and captures stay local to their
alternative. Invalid tags or active payloads stop with the declared conservative
fallback. Cancellation instruction, call and deferred-use chains preserve their
previous short-circuit order; completion retains an exhaustive decision where
its outcomes require different follow-up evidence.

## Call-use effects

`calls/rules/call_effect_uses.rules` classifies direct effects and the exact transparent
alias forms. Its native result contains effect bits and an optional alias;
`CallEffects` retains call/closure dispatch, every-use traversal, visited state,
budgets and memoization. The alias result is one step, not a recursive proof.

Stores distinguish mutation of the supplied address from retention of the
supplied value, including their combination. Publication through a channel is
asynchronous and incomplete; map keys and values retain the original uncertainty.
Loads read the slot without following its pointee. Unsupported operations remain
unknown rather than being inferred harmless.

`calls/rules/call_field_effects.rules` selects the matching embedded field projection
and classifies loads, whole-owner writes and publication. The Go caller rebases
the selected child path and resumes the same bounded walk. Other fields cannot
replace the selected slot. A whole-owner write still includes that slot.

`call_effect_uses_test.go` exercises these policies through the public value and
field-call proofs on compiled SSA, including wrappers, merged aliases, local
spills, synchronous/deferred/launched calls, capture loads, nested field paths,
and recovery with a fresh allowance after exhaustion. The existing metadata
allowance tests continue to cover late matching arguments and captures.

## Goroutine instruction labels

`goroutineownership/rules/entry.rules` preserves the precedence of a stored testing
receiver, an exact selected receive and a possible selected receive before an
instruction's own effect. `classify.rules` handles ordered receive evidence and
channel/map publication. All identity, containment and selected-case questions
are explicit queries; no counted search is a speculative guard.

The enclosing instruction cache and trace boundary are unchanged. Go dispatches
returns and stores to their existing proof functions and calls through the
contract stages below. Standalone receive
recognizers live under the analyzer's `receives` package, while spawn-specific
identity remains with its classifier. Query phases live beside spawn budgets.
No traversal, budget accounting, callback binding or flow query moves into rules.

The classifier's compiled-SSA tests preserve action/reason pairs and cache reuse,
including a send at a selected receive's arm entry. Full analyzer fixtures retain
mixed select, alias, ownership-transfer and budget controls. Return/store proofs
and callee traversal retain their existing Go implementations.

## Goroutine call contracts

`call_prefix.rules` handles missing calls, launched observers, builtins and direct
joins before any summary or retained-owner query. A launched builtin keeps the
opaque-launch policy. Builtins other than append return a recognized `none`;
append asks whether its arguments consume tracked values.

`call_contracts.rules` orders retained-owner participation, pipe peers, concurrency
summaries, WaitGroup bookkeeping, testing cleanup and gomock return contracts.
Each query executes once when reached. A recognized `none` from bookkeeping or
cleanup stops the chain; an unrecognized answer proceeds. The gomock argument
query runs only after its registered library contract matches. Unavailable or
invalid query-domain outcomes terminate with an opaque call label.

The targets share the existing `helperCallProof` label and native recognition
boolean. Direct joins, pipe peers and summaries produce recognized labels at the
same proof boundary rather than needing request or response wrapper structs.
Callee resolution, recursive helper binding and opaque-call consumption remain
Go queries. The instruction cache still surrounds the complete call decision.

The analyzer-local `containment` package owns possible possession, including old
stores and mixed choices; it never claims exact handoff. `origins` owns bounded
factory/registry provenance. Native transfer and lifetime searches consume
those same identity and provenance queries. Their moved searches and rationale comments retain the
same transparency, storage and cutoff rules.

Compiled-SSA label tests cover launched builtins, append, other builtins, possible
group receivers and direct waits. Contract tests pin terminal `none` for group
bookkeeping and testing cleanup. Existing summary and retained-owner tests keep
candidate attribution, cutoff precedence and fresh-cache recovery controls.

`proofpolicy/rules/caller_lifetime.rules` sequences the four caller-lifetime bounds
under one native allowance. Each observation retains its witness and post-query
cutoff; unavailable evidence returns the original attributed cutoff proof before
considering a partial positive result. `proofpolicy/rules/edge.rules` chooses exact
selected receives, counted drains, possible receives and retained-context
ownership in order. A counted drain's exact or unknown action is terminal.
Native query methods retain the original trace timing and keyed edge evidence.

`containment/rules/aggregate.rules` extracts only the aggregate operand of the admitted
load/field/index/lookup forms. Wrapper peeling, iteration and per-step charges
stay in the native root walk. `completionsteps/rules/notification.rules` selects send
and close notification evidence for all three existing callers. Only sends ask
the native terminal-coverage callback; a launched notification is excluded before
any query, and an ordinary close in a cycle remains per-item cleanup. Deferred
closes retain the caller's registration-coverage requirement. These policies
reuse the existing analyzer-owned packages without adding root production files.

## Process ownership actions

`action_entry.rules` asks possible-handoff and deferred-wait queries first.
Uncertainty there stops the action, preserving the original early boundary.
`action_settlement.rules` recognizes direct waits and process release before the
lifecycle completion/transfer query. Go prepares the same completion allowance
between these stages; direct-witness identity queries retain their independently
owned allowances and do not spend the prepared completion budget.

An unfinished lifecycle search supplies a provisional unknown fallback rather
than ending the action. `action_remainder.rules` permits an exact external-field
transfer, handle completion or process exit to override it, in the original
order. A handle cutoff remains unknown unless the later exit witness wins.
An absent handle preserves the earlier fallback. Non-budget lifecycle uncertainty
retains its original distinction from an abandoned search.

The existing lifecycle request still owns completion, transfer modes, selected
summary masks and receiver-store evidence. Native function signatures carry the
state, recognition flag, allowance and provisional fallback directly. Handle
argument scans and the instruction/command cache remain the same Go paths.

`action_order_test.go` pins baseline outcomes at zero, small and full allowances
on compiled SSA, including early uncertainty and late positive evidence. Separate
remainder cases distinguish a truly exhausted handle query from an empty unused
allowance and check fallback retention. Existing action-cache and analyzer tests
retain command identity, fresh-candidate and flow controls.

`processownership/rules/prestart.rules` selects prefix availability, external/helper
ownership, prior registration and successful-return evidence. Native methods
create child budgets only when reached. `return_admission.rules` replaces the
per-return closure table with one selected reason; the flow emits its evidence
event once. `handoff.rules` orders possible Wait participation while native
queries retain identity, reachability, completion and callback-argument scans.
Its uncertain completion retains the original reason, Method and Provenance.
The immediate noncall/Wait/visible synchronous cases retain their unspent-budget
behavior even when the caller supplies an already exhausted allowance.

## Resource lifetime instruction classification

`resourcelifetime/rules/classify.rules` owns the ordered instruction policy. Collection
and result-guard labels stop the chain even when they select no action. Compressor
abandonment, SQL parent cleanup and transaction cancellation precede release
recognition; ambiguous cleanup, paired-error cleanup, loop-only release and opaque
consumption retain their later positions. Each reached evidence query runs once
with its original child allowance and candidate pool. Unknown is never promoted
to settlement, and exhausted evidence cannot fall through as an absent cleanup.

Selections can share an answer across complete evidence variants with
`(Proven (reason: reason) | Unknown (reason: reason)) <- queryName ...`
with an explicit else arm for remaining outcomes. A whole-result bind
retains the exact native release tuple, so the action/reason pair needs no new
adapter record. Bindings continue to delegate to existing proofs, caches and flow.

The analyzer-owned `sqlparents` helper owns exact SQL parent relationships and
storage child limits. The enclosing analyzer maps those relationships onto its
existing reason vocabulary and uncertain-lifetime policy. Prior-defer discovery
and instruction classification consume that same proof. No parent-close contract
becomes a synchronous child-release guarantee.

`classify_order_test.go` pins cleanup, observation, conditional cleanup, loop-only
cleanup and early collection precedence against the handwritten classifier's
baseline. Existing budget-observer, SQL identity, result-guard, ownership and
wrapper regressions continue to guard the underlying proof boundaries.

## Static result guarantees

`resultfacts/rules/leaf.rules` classifies the original SSA value after the existing
leaf allowance and call-result summary path. Nil and Boolean literals retain
priority; concrete interface boxing is nonnil even for a typed nil payload.
Interface/type-parameter payloads remain unknown. Allocations, channel/map/slice
creation, functions and closures retain their original nonnil guarantee.
Numeric constants, parameters, unresolved projections and other forms retain
unknown. The recognizer does not infer guarantees through memory.

The existing reaching/storage fold, point-in-time content queries, imported
facts, recursion, summary cache, publication cutoffs and normal-return policy
remain in Go. No wrapper record or query adapter is introduced: the generated
native function returns the existing `Guarantee` enum. Its three pure hooks
only inspect literal and type structure.

[Result guarantee tests](../../internal/analysis/passes/resultfacts/results_test.go)
cover the recognized forms alongside typed-nil boxing,
abstract generic boxing, Boolean parameters, scalar constants and slice
projections. The added cases pass against the handwritten baseline. Existing
allowance, storage, relation, cross-package and publication regressions remain
authoritative for the surrounding summary engine.

## Heap value definitions and leaves

`heapmodel/operations` recognizes alias, field/index projection, slice, load and
opaque definitions. Its plans retain exact SSA operands; the enclosing graph
applies every load and projection against the current region state. Transparent
wrappers use the existing form mask, and comma-ok assertions are left to their
extractions. Unsupported definitions do not gain storage or identity evidence.

The separate leaf target distinguishes external values, nil aggregate slots,
opaque callable values and pending local instructions. A pending local value
retains an empty pointee set until the fixpoint reaches it; a foreign instruction
retains unknown. Type tracking, owner checks, caching, joins, escape effects and
fixed-point scheduling remain in Go. Existing region storage parity, alias,
projection and aggregate tests exercise the consuming graph rather than merely
checking the generated plan shape.

## Completion outcomes

`lifecycle/completionpolicy/rules/outcome.rules` selects the existing proof after a
completion search fails to prove its target. Budget exhaustion precedes loop
uncertainty, which precedes incomplete nested work, then a completed negative
search. Body availability supplies local SSA provenance even at a cutoff. An
unavailable body has no such provenance. The enclosing request still observes
exactly the returned reason with the original launch position and details.
Search, recursion, positive witnesses and allowance spending remain in Go.

## Capture and producer recognition

Capture root rules select one identifier or next expression through parentheses
and map indexes. The caller owns the walk; arrays, slices and selectors retain
their existing conservative boundary. Channel statement rules recognize direct
sends and standalone receives. The caller retains statement ordering, exact
channel-object identity and pending-send updates; recognition does not claim
semaphore capacity or race freedom.

Producer receiver rules dispatch helper calls before direct receives and select
states. Helper outcomes retain their count, uncertainty and reason intact, and
alias queries execute only for reached receive forms. Select counting retains
the existing may-alias policy. Block traversal, partial-count accumulation,
cycle uncertainty and shared helper allowances remain unchanged.

`producerlifecycle/rules/abandoned.rules` sequences the normal-return witness,
producer count, receive availability and count comparison. Unknown producer
proofs return before receiving queries. `helper_receive.rules` keeps launch and
builtin exclusions ahead of summary lookup, then sequences complete summaries,
nonreceiving worker evidence and opaque consumption with the same allowance.
`receive_schema.rules` shares native count/uncertainty/reason storage across
these policies and instruction recognition. An unavailable helper unrelated to
the channel keeps its false uncertainty payload and helper-unknown reason.

## Capture guards and exclusive-lock transitions

`concurrentcapture/rules/mutation.rules` owns the authoritative guard order after the
existing ordered lock proof. A known held guard precedes the unknown-lock syntax
fallback; worker and channel queries remain lazy in that order. The rules return
the existing mutation proof, so reporting and tracing keep one decision path.

`lock_transition.rules` selects a complete transition before the enclosing region
changes its held set. Projections, indirect resources, absent mutex identities,
unmatched releases, condition waits and read locking retain uncertainty. Only a
reached exact unlock asks whether its resource is held. Unknown remains sticky
while later exact effects still update the held set. Prefix completeness, summary
queries and all map updates remain in Go.

## Lock write and index recognition

`lockorder/lockforms` selects the exact destination of a store, map update or
supported builtin. Copy selects only its destination, and application-defined
lookalikes cannot pass the SSA builtin matcher. Store destinations are direct;
container destinations retain the caller's separate loaded-owner evidence.

Owner-address rules admit fields and constant index addresses. The existing
identity/alias walk remains in Go, and dynamic indices retain their conservative
stop. Only a slice source may expose its backing array through a load; a loaded
pointer stays another object. Dynamic mutex-index rules return either a terminal
answer or one source for the existing reaching walk, preserving its form mask,
allowance and termination state.

## Lifecycle structural leaves

`lifecyclefacts/lifecycleforms` supplies exact response Body field addresses and
shared handoff-use plans for direct and wrapped results. A Body match requires
the original load form, exact Response type and Body field contract; it establishes
neither acquisition nor unchanged contents. Handoff recognition permits only the
original box/interface/phi forms and equality comparisons. Calls request cleanup
evidence in the walk for direct results and exact wrapper evidence in the walk
for retained results. The latter still
requires positive wrapper depth at a return and rejects calls at its depth bound.
Unsupported casts, stores and captures retain the conservative rejection.

The enclosing pass retains result-position agreement, visited state, cleanup
ordering, imported facts and ownership publication. Existing owned-result tests
now also pin comparisons and a deliberately opaque retyping, alongside fresh,
borrowed, retained, previously closed and callback-owned results.

`lifecyclefacts/rules/returned_view.rules` sequences unchanged-return, possible-release
and parameter/result cleanup-capability queries in their original order. Type-only
view evidence wins before receiver-method completeness is needed. The final
stored-field query requires complete method knowledge before scanning; an
unavailable method cannot establish that a field is never released. Field
collection and lifecycle/type searches stay in Go.

`lifecyclefacts/rules/evidence.rules` keeps local and imported source precedence
together; `selected_mask.rules` owns mask selection. Native facades retain their
original signatures. The generated policies return the selected whole proof
with a decision flag; optional-request binding, local searches and fact lookup
remain native. Supplied positive proofs stop before later queries. A positive
transfer can settle an abandoned completion; failed or unknown transfer cannot
erase that completion's cutoff. Imported masks still order direct discharge,
ordinary identity/containment, immutable capture, opt-in projection, conditional
cases and possible-alias uncertainty. The wrapper emits the existing event once.

The native evidence file stays cohesive around request binding, proof carriers,
query adapters and observation, slightly above the 400-line review threshold.
Direct and wrapped fresh-result searches share `result_handoffs.go`; their
independent cleanup, wrapper, depth and result-slot checks remain unchanged.
Invocation-only and complete summary inference also share their existing
pass-owned callback engine in `summarize.go`. Both file consolidations preserve
the original bodies and rationale comments. This left the root package at
20 production files when generated code still counted toward the limit.

Mask selection has a separate generated entry point so the imported policy
does not call back into its own dispatcher. During dogfooding, that static
cycle caused heap-region construction to wait for its own in-progress graph.
The separate entry point preserves reached-query behavior and avoids the cycle.

## Lifecycle retention and escape uses

Lifecycle retention rules distinguish loose retention, strict storage and
coverage on every return. Captures, sends and map updates cannot establish an
every-return handoff. Strict retention leaves returned owners to their own
summaries. Recognized stores, closures and calls request the existing proofs;
the mode gate is part of the generated classification rather than a parallel
handwritten structural policy.

Escape-use rules distinguish in-place invocation/defer from passing the value
to another callee, follow exactly the existing conversion/projection forms and
leave store destinations to the host's local-storage proof. A store into the
value itself is not publication. Unrecognized uses retain the original escape
fallback, while unsupported retention instructions retain no witness. Recursive
use walks, budgets, memoization and strict/loose imported-summary interpretation
remain in Go, preserving the opposite proof polarities of those two queries.

## HTTP acquisition evidence

The analyzer-owned `resourcelifetime/httpacquire` helper owns exact HEAD request
provenance, local endpoint/client identity and bounded writer/default-effect
searches. Its three rulesets select HEAD outcomes, endpoint admission and empty
framing effects. HEAD recognition checks construction before mutation and client
evidence. Local endpoints spend their existing visit before server and handler
resolution; caller defaults, handler defaults and writer framing retain that
order under one effect-child quota. The final allowance fence remains outside
the generated policy, so an interrupted decline cannot become normal leak flow.

The parent analyzer projects the helper's protocol reasons once onto the existing
resource reasons. Its trace positions, considered/evidence outcomes and cleanup
flow remain unchanged. Existing HTTP controls continue to test both the extracted
memos and the complete consuming flow, including root-only default Do allowances,
strict helper memo isolation, child cutoff and fresh recovery. These searches
retain all existing form masks, identities, visit charges and rationale links.

## Acquisition error decisions

`acquisition_absence.rules` orders exact sentinel equality, successful type
assertion, `errors.Is`, exact `errors.As`, inferred error predicates and the
existing legacy API table. An inferred predicate's budget cutoff is terminal;
an unavailable predicate may continue to the legacy contract. This distinction
is explicit in the native recognized/unrecognized query outcome. The structural
call rule skips only the equality test that cannot apply to a call.

`acquisition_branch.rules` consumes the absence proof before testing Testify's
NoError contract or the ordinary success-edge query. Unknown remains terminal.
A recognized NoError answer requires the same availability fence; otherwise the
ordinary edge query retains its conservative cutoff behavior. The generated
function returns a trace flag beside the existing branch proof. Go emits a trace
only for the same accepted guards, after selection, and still owns the initial
visit, branch admission and proof's success-bit availability.

## Completion launch and callback recognition

`completionpolicy/rules/launch.rules` selects synchronous calls, defers, goroutine
launches, exact WaitGroup.Go callbacks and testing Cleanup registrations. The
plan carries the original call/callback and launch kind into the existing body
resolver. Only an actual defer permits OnceFunc unwrapping: an immediate call
could already have consumed the wrapper. Merely storing or constructing a
callback does not create a launch.

`callback.rules` supplies exact closures, permitted OnceFunc sources or a storage
query address. The existing reaching fold, stable-content observation point,
child allowance and complete-callee requirement remain in Go. The launch kind's
reason projection moves with that vocabulary and keeps the same evidence codes.
Existing completion boundaries, callback replacement, body coverage, case/memo
isolation and all-allowance regressions remain the behavioral contract.

## HTTP use contracts

Seven companion policies own client provenance, zero-client-cell uses, request
origins and uses, local server uses, writer uses and visible default effects.
The consuming Go loops still charge each referrer, operand and instruction visit;
reaching folds own recursive request state and memoized helper queries retain
all child quotas. Pure symbol and shape hooks cannot spend those budgets.

Client-cell state accepts one fresh stored allocation, direct Do loads and
verified captures; debug uses preserve the current stored flag. Request origins
admit only result zero of exact constructors and recognized rebinding calls.
Origin recursion precedes immutable-use checking, and the two folds never share
a visited set. Request Header edits, local URL loads and exact server clients
retain their existing separate use searches.

Writer status decisions are terminal even when a status is rejected; they never
fall through to helper inference. Root-only default-client Do reads can skip the
operand scan only after the original use proof succeeds. Failed use proofs still
charge the operand scan, while helper summaries stay strict. Existing HTTP
allowance, child-cutoff, memo-isolation and complete-flow tests exercise these
production paths.

## Concurrency summary representation

Effect collection support generates both passive admission and schema-derived
branch snapshots in one mechanical output; the policy and snapshot schemas stay
in separate source files. `cloneEffects` remains the single branch-copy API.
Nested hooks apply generated worker/choice/arm copies to detached lists, while
operation alternatives retain their original outer-and-inner copying depth.
The operation metadata, condition contexts, worker condition arms and pre-proven
path summaries remain borrowed exactly as before. Nil and non-nil empty lists
remain distinct. Collection, cutoff tracking, source folding and branch joins
remain Go responsibilities; derived copying does not define summary equality.

## Worker containment, helper escapes and receive sites

`goroutineownership/containment` generates the SSA cases selected by the existing
possible-possession walk. Alias checks still precede leaf recognition; closure
bindings, old stores and sibling selections retain their original traversal and
visited state. An asserted owner can carry a stored target, but its assertion is
not a transparent transfer. Generated stored-use cases retain exact store-address
comparison before following a value.

Helper escape rules pass native predicate parameters to explicit queries. The
caller retains the tracked-kind policy, recursive helper memo and allowance;
there is no query-host adapter or second helper cache. A returned owner handle
or offered select send remains unknown participation, not positive completion.

`receives` generates direct, selected and channel-range cases, together with
ordered guaranteed-receive decisions. Selected arm-entry evidence precedes local
instruction evidence. A select that is mixed, empty or nonblocking remains a
terminal negative for guaranteed completion, even if some offered receive uses
the tracked channel. Select enumeration retains its original per-arm spending,
and channel predicates retain the caller's exact/possible identity policy.
Existing receive identity, select-edge, helper-use and worker-flow regressions
exercise these paths through the analyzer.

## Lock instance identity and iteration dependence

`lockorder/lockidentity` owns one instance-identity walk and its exact getter and
field contracts. Generated leaves choose getter, field, indexed and named SSA
cases. An unresolved call or field is terminal: it cannot fall back to an SSA
name and invent a stable lock across executions. Indexed identity still resolves
the owner before the index, including when the owner is unknown. Storage queries,
walk marking and the final child/pool fence remain in Go. The parent class and
callee-path code use the same getter and field helpers.

`lockvariant` generates iteration-dependent origin and extraction cases while
sharing the caller's existing reaching walk. Receives outside a cycle are
terminal false; select index/receive-ok metadata never borrows payload dependence
or incurs another tuple visit. Index-before-owner and left-before-right queries
retain their short circuiting and common visited set. Calls follow their arguments
in order, and transparent wrappers use the shared caller-selected form mask.
No iteration-dependent source proves distinct runtime locks: it supplies only
uncertainty. Existing lock identity, copied-receiver, loop and all-allowance
controls remain the behavioral contract.

The main held-state transfer remains Go orchestration. It interleaves trace and
map updates, deferred-release registration, caller-ownership checks, and exclusive
acquisition queries. Moving the whole transfer into a pure builder would violate
that order; a separate rule dispatcher would leave the same state machine in Go.
The substantial finite source decisions it consumes are generated instead.

## Conditional result policy

Result inference uses generated decisions for assumed Boolean leaves and the
relationship between a result and its paired error. The reaching fold, exact
forwarded-slot decoding and callee summary memo remain Go queries. A vacuous
return does not query its result or a forwarded summary; a decided assumption
returns before the cutoff fallback, while an undecided cutoff never invokes
unconditional leaf inference. Witness and hold remain separate outputs.

The summaries provider also uses generated comparison decisions. It queries the
original value before structural recognition, preserves literal-left swapping,
and queries paired-result evidence only when unconditional equality is unknown
and the comparison literal is nil. Negation retains the Boolean payload even
when unknown; only a known answer can prune flow. The fixed-argument recognizer
retains caller bindings before construction evidence, including a bound
`OutcomeAny`, and preserves typed-nil interface boxing. These policies share no
new cache or allowance and do not broaden the proof model.

## Path-guard evidence

The guard rules keep loaded identity distinct from stable Boolean identity.
Loaded recognition runs first. Stable equality then queries its left operand
before its right operand, and an unavailable operand continues with the
existing computed-Boolean cycle query. That continuation shares the consumed
allowance; no failed strategy refunds work or tries the right operand early.

Address rules return a root or projection step. Go spends before each step,
recurses over the selected source and applies the rendering memo afterward.
A returned pointer remains one address only until the call reruns; the existing
invalidation walk still owns that boundary. Loaded rules call the same address
and rendering query, including when the address is unavailable. They preserve
its string and polarity payload while returning availability false; unresolved
addresses never populate either memo. A warm memo still pays for resolution.

Literal comparison keeps ordered operand decoding and both cutoff fences in
Go. Rules select nil equality or a supported constant operator only afterward.
A nil pair never reaches `constant.Compare`; missing literal payloads and
unsupported operators remain undecided. Existing path, execution-oracle and
local/shared budget tests cover the consuming flow, and the loaded-format test
pins the nonempty unavailable payload.

## Callback value and context policy

`completionpolicy` recognizes retained capability separately from exact
callback identity. A generic call's arguments may carry a callback without
promising its invocation; this remains broader than the exact `OnceFunc`
contract. The existing any-origin fold and storage/callee queries consume the
selected source or argument list without a new walk or allowance.

Binding rules distinguish direct functions/closures, field loads, element
loads, ordinary dereferences and stable cells. Go still substitutes invocation
environments and transparent wrappers before selection, and preserves the
observation point in each storage query. Whole-slice aggregate views require
all three bounds to be absent. Enclosing reference rules intentionally use a
different contract: only field or local-allocation loads can resolve; other
loads and unresolved parameters, free variables and phis remain unavailable.

Context admission rejects launches, sends and callback-bearing boxes, maps,
returns and nonlocal stores before collecting invocation frames. Its bounded
type/address predicates retain their original limits and conservative polarity.
Conditional comparison rules prefer an existing right constant, swap only when
it is not constant, and either select error nilness or return a Boolean operand
for the existing recursive decoder. The exact operand identity is covered by a
regression; callback, enclosing-frame, field/element and cutoff fixtures remain
the consuming tests for these generated policies.


## Lock-state copy and transfer sequencing

`lockorder/rules/copy.rules` sequences the existing charged slice, guard and origin
copies. An exhausted copy exposes no partial successor; nil and nonnil empty
slices remain distinct, and copied maps are writable even from nil inputs.
`transfer.rules` distinguishes direct/deferred release, deferred acquisition,
and acquisition. Registration precedes dynamic-receiver and caller-ownership
checks. Preparation precedes ordering proof, and a cutoff cannot publish the
new held lock or origin. Alias searches and collection updates remain Go queries.

## Concurrency resource-reference policy

`concurrencyfacts/recognition/rules/reference.rules` selects storage-resolved symbolic
resources, standard condition constructors and mutex addresses. Cancellation
provenance is still queried first; an unrecognized call retains its SSA value
alongside false availability. Root-binding, fixed-load and embedded-path rules
keep their distinct unary and global-address contracts. Fixed loads require
dereference, while global channel contents never inherit mutex-address identity.
The inert-builtin rules require SSA builtin identity and retain the original
operand checks; ordinary functions with matching names cannot enter that table.

## Completion mapping and result forwarding

`lifecycle/completionmapping` owns capture, argument, deferred-cell, receiver and
path-translation policies. The enclosing search supplies callback/containment
queries, invocation environments, memo effects and binding iteration. All phases
share one mapping representation for exact values, projections, owners and
callbacks. Failed exact mappings retain their original payload. A selected
captured-owner case that fails storage proof cannot retry aggregate containment.
Alias uncertainty and independent child cutoffs retain their different memo
behavior, and deferred cells are read at invocation time.

Receiver and translation rules reject sibling paths and further projections of
strict projections. A nil owner path is an already-matched direct receiver and
needs no new query; a nonnil empty path remains a nested answer requiring mirrored
identity. Possible derivation after an unavailable receiver path supplies only
uncertainty, never must-complete coverage.

The conditional-return targets select a result, resolve only Boolean spills,
and exclude contradictory outcomes before accepting earlier completion. Error
interface boxes remain intact, including typed nils. Forwarding plans preserve
the original call and result slot; recursive exact-local coverage stays in Go.
An absent result remains uncovered even when earlier completion was recorded.
The mapping and return tests pin these ordering, payload and cutoff boundaries;
existing consumer fixtures retain the end-to-end checks.

## Goroutine ownership proof policy

`goroutineownership/proofpolicy` owns the stable proof/reason domain and final
ordered decisions. Discovery availability precedes all queries. Lifecycle and
pre-spawn evidence precede the single obligation walk, while guarded exact flow
precedes opaque flow. Flag, loop-count, shared-storage, buffering, signal-census
and process-entry evidence retain their original order and conservative outcomes.
A decided unknown result stops later strategies just as a proven result does.

The consuming analyzer executes each reached search and records a negative
considered reason before the next query. Flow evidence is evaluated once and
reused; cutoff packets retain their exact child allowance for the original
observation phase. No trace-only result list is constructed when tracing is off.
Caller-lifetime enumeration remains its existing declarative Go table, and all
instruction classification, cache, traversal and reporting boundaries remain.

## Local completion evidence and path agreement

`lifecycle/rules/completion_local.rules` selects direct callback invocation, nested
callback completion, exact target invocation, and ordinary nested completion in
that order for each mapped local. Callback failure never falls through to the
ordinary target search. A nested owner or contained target must name the exact
path; an unknown path may settle an ordinary exact/projection mapping but cannot
settle a sibling field. The caller retains the ordered local loop, instruction
budget charge, incomplete flags, memo invalidation and final path observation.

`completionmapping.Paths` owns path agreement across completing instructions and
callees. Its generated fold preserves the first path payload and marks conflicting
or unnamed evidence inexact. Later matching observations cannot restore lost
exactness. An unseen callee adds no observation; a nil collector records none.
The existing translation rules map nested paths before they reach this fold.

## Resource opaque consumption

`resourcelifetime/resourcepolicy` owns the shared resource reason/evidence domain
and ordered instruction, storage, send, call-dispatch and named-function decisions.
Queries retain the analyzer's allowance limits, caches and source observations.
The callback boundary precedes asynchronous exposure, which runs only for carried
arguments. Constructor-chain evidence still precedes the uncarried-call exit;
aggregate containment reaches result-publication evidence only when proven.

Unknown evidence stops the selected decision, while false answers retain their
original reason payload. In particular, a non-consuming builtin remains
`Appended/false`, a dynamically chosen non-carrying function remains
`DynamicCallee/false`, and a readable or summarized static callee can remain
`UnsummarizedCallee/false`. Those values describe an inspected alternative, not
numeric action codes. Sends and select arms share one handoff/payload policy;
arm iteration, storage caches and recursive searches stay in Go.

Closure decisions use the same domain and explicit query boundary. Deferred-cell
cleanup precedes guarded-body evidence, which retains its separate 1,000-step
allowance. Aggregate-owner evidence precedes the optional capture search; a
carried argument skips capture, while absent capture prevents even a launched
literal from becoming an opacity label. Started literals stop before body
queries. For local literals, unreadable-callee evidence precedes retention;
negative retention still returns `CapturedByRetainingLiteral/false`. The existing
capture, storage and lifecycle searches remain authoritative Go queries.

## Resource final evaluation

`resourcepolicy` sequences the memory-writer exclusion, acquisition evidence,
correlation, initialization, flow and final exclusions. `Begin` is reached only
after the exclusion fails, so it alone binds candidate tracing and the allowance.
Queries return explicit phase contexts; no later query relies on a field filled
by an earlier query. Go retains correlation storage and resource-phi binding at
initialization, which cannot run until correlation availability is established.

The paired error's exact value survives a completed negative lookup. A
non-reportable flow returns intact without querying exclusions. A reportable flow
queries database cleanup before pool availability, then evaluates one process-exit
child whose positive result precedes its cutoff. Only an available, unexcluded
flow retains its original diagnostic witness. The graph walk, owner discovery,
classifier caches and effect application remain in Go.

Preparation sequences dominating-defer evidence, complete owner discovery,
collection/guarded-defer initialization and prior cleanup before the path walk.
A positive prior-cleanup observation commits immediately in its Go query; the
rules then return uncertainty. Partial owner discovery supplies no classifier
inputs, and guarded-defer uncertainty stops before prior-cleanup evidence.

## Lock final proof selection

`lockorder/lockpolicy` sequences missing-release evidence and the decisions for
one matched read-lock/write owner. The analyzer retains bounded private-lock and
caller-contract queries, field/destination evidence, possible-writer searches and
candidate iteration. Census and contract packets retain availability alongside
their exact reason; a shortened search cannot override cutoff with a witness.

Read-lock decisions return an explicit continue/stop step together with the
proof. Ordinary uncertainty can be replaced by a later candidate; writer cutoff
and reportable evidence end the search at the current identity. When no candidate
reports, the last matching candidate's proof and identity survive. The root's
reason aliases retain all original numeric values and trace strings; reporting,
held-state flow and diagnostic evidence buffering remain unchanged.

## Worker completion instruction plans

`goroutineownership/completionsteps` recognizes the distinct structural plans
for terminal completion and the remaining suffix of an opaque call. Return,
jump and RunDefers forms belong to terminal completion. An opaque-call suffix
also permits DebugRef, dereference loads and exact builtin close calls. Receives,
other unary operations, named close functions and new launches remain work.

The plans leave CFG scheduling, cycle rejection, allowance charges and the
completion-only defer census in Go. Terminal RunDefers still requires the
original WaitGroup Done origin. A plan never proves a deferred callback safe;
only the reached existing census can do so. The pure recognition adds no query
charges and does not broaden either mode's accepted instruction set.
