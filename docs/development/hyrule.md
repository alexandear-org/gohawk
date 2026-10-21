# Hyrule integration

Hyrule generates structural recognizers and ordered decision functions from
rules. Runtime builds compile the generated Go; analysis does not invoke the
compiler or interpret rules. Generated Go and the ownership manifest are ignored
by Git, so a fresh checkout must generate before compiling.

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
standalone GoHawk module, including its supported Go floor. The manifest
records generated-file ownership and hashes. Edit rules or bindings and
regenerate, never patch generated Go.

## Migration scope

A strong fit is a substantial ordered structural recognizer, a decision over
explicit evidence, or a finite state-transition policy. Traversal, alias search,
fixed-point scheduling, budgets, caches, and effect application remain in Go.
A trivial enum switch or an already-declarative lookup table is not a migration
requirement merely because a rule could express it.

The repository-wide migration covers these families. Integration is complete
only when the consuming production path uses the rules and the old decision
path is removed; generator support and examples alone do not qualify.

| Family | Current integration |
| --- | --- |
| Select-index comparison recognition | Generated shared recognition used by summary dispatch and selected-receive edge evidence |
| Standard synchronization primitives | Generated shared recognition used by concurrency call summaries and heap mutex effects; count acceptance remains summary policy |
| Concurrency inert-data and passive-instruction admission | Generated admission over inert-data contracts and budgeted field-path evidence; collection stays in Go |
| Concurrency context/callback operation recognition | Generated context calls/origins and callback input/call recognition; storage resolution, callback binding and summary construction stay in Go |
| SSA wrapper, projection, and structural source families | Generated transparent wrappers and ownership-source recognition; other projection/source families remain |
| Call-use and heap-operation classification | Generated direct call-use effects, field-use admission, heap value definitions and static heap leaves; memory effects, recursive searches and state application stay in Go |
| Lifecycle completion/transfer recognition and outcome selection | Generated unproven-completion outcomes; completion/transfer recognition remains; recursive searches stay in Go |
| Goroutine ownership instruction/call labels | Generated arm-entry, receive, publication and ordered call contracts; return/store proofs, callee binding, caches and searches remain in Go |
| Cancellation ownership instruction/call labels | Generated instruction, direct/deferred, call-contract, exact-argument and return decisions; owner discovery, searches, cache and flow stay in Go |
| Process ownership decision selection | Generated post-Start return policy and ordered instruction ownership actions; lower-level handle scans, caches and flow remain in Go |
| Resource lifetime classification and acquisition-boundary patterns | Generated ordered instruction classification; acquisition-boundary patterns remain; catalog collection stays a table |
| Defer-in-loop resource classification | Generated per-value and per-acquisition decisions; call discovery and loop traversal remain in Go |
| Lock operation, write-target, and held-state transition families | Generated write destinations, owner-address steps and dynamic-index leaves; main held-state transition policy remains; identity and cycle search stay in Go |
| Concurrent capture syntax and guard decisions | Generated mutation-root steps, channel statements, guard decisions and exclusive-lock transitions; prefix collection and effect application stay in Go |
| Producer lifecycle use classification | Generated receiver instruction classification; helper inference, binding scans, counting and cycle policy stay in Go |
| Result and lifecycle summary leaf recognition | Generated static result leaves, response-body leaves, freshness, retention and escape-use classification; summary inference and reaching walks stay in Go |

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

## Concurrency admission

`concurrencyfacts/admission/data.rules` owns inert-data instruction admission.
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

The `concurrencyfacts/recognition` package owns four native recognizers. Context
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

First-answer policies use `query ... select Pattern => answer <|> ... <|>
pure fallback`. Each query is explicit and lazy; unselected valid outcomes
continue inside the committed rule. Answers and captures stay local to their
alternative. Invalid tags or active payloads stop with the declared conservative
fallback. Cancellation instruction, call and deferred-use chains preserve their
previous short-circuit order; completion retains an exhaustive decision where
its outcomes require different follow-up evidence.

## Call-use effects

`calls/call_effect_uses.rules` classifies direct effects and the exact transparent
alias forms. Its native result contains effect bits and an optional alias;
`CallEffects` retains call/closure dispatch, every-use traversal, visited state,
budgets and memoization. The alias result is one step, not a recursive proof.

Stores distinguish mutation of the supplied address from retention of the
supplied value, including their combination. Publication through a channel is
asynchronous and incomplete; map keys and values retain the original uncertainty.
Loads read the slot without following its pointee. Unsupported operations remain
unknown rather than being inferred harmless.

`calls/call_field_effects.rules` selects the matching embedded field projection
and classifies loads, whole-owner writes and publication. The Go caller rebases
the selected child path and resumes the same bounded walk. Other fields cannot
replace the selected slot. A whole-owner write still includes that slot.

`call_effect_uses_test.go` exercises these policies through the public value and
field-call proofs on compiled SSA, including wrappers, merged aliases, local
spills, synchronous/deferred/launched calls, capture loads, nested field paths,
and recovery with a fresh allowance after exhaustion. The existing metadata
allowance tests continue to cover late matching arguments and captures.

## Goroutine instruction labels

`goroutineownership/entry.rules` preserves the precedence of a stored testing
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
factory/registry provenance. Candidate transfer and lifetime decisions consume
those same queries in Go. Their moved searches and rationale comments retain the
same transparency, storage and cutoff rules.

Compiled-SSA label tests cover launched builtins, append, other builtins, possible
group receivers and direct waits. Contract tests pin terminal `none` for group
bookkeeping and testing cleanup. Existing summary and retained-owner tests keep
candidate attribution, cutoff precedence and fresh-cache recovery controls.

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

## Resource lifetime instruction classification

`resourcelifetime/classify.rules` owns the ordered instruction policy. Collection
and result-guard labels stop the chain even when they select no action. Compressor
abandonment, SQL parent cleanup and transaction cancellation precede release
recognition; ambiguous cleanup, paired-error cleanup, loop-only release and opaque
consumption retain their later positions. Each reached evidence query runs once
with its original child allowance and candidate pool. Unknown is never promoted
to settlement, and exhausted evidence cannot fall through as an absent cleanup.

Selections can share an answer across complete evidence variants with
`select (or (Proven reason:$reason) (Unknown reason:$reason))`. A named selection
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

`resultfacts/leaf.rules` classifies the original SSA value after the existing
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

`lifecycle/completionpolicy/outcome.rules` selects the existing proof after a
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

## Capture guards and exclusive-lock transitions

`concurrentcapture/mutation.rules` owns the authoritative guard order after the
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
freshness-use plans. A Body match requires the original load form, exact Response
type and Body field contract; it establishes neither acquisition nor unchanged
contents. Freshness recognition permits only the original box/interface/phi
forms and equality comparisons. Calls request the existing cleanup proof rather
than becoming accepted uses by recognition alone. Unsupported casts, stores and
captures retain the conservative rejection.

The enclosing pass retains result-position agreement, visited state, cleanup
ordering, imported facts and ownership publication. Existing owned-result tests
now also pin comparisons and a deliberately opaque retyping, alongside fresh,
borrowed, retained, previously closed and callback-owned results.

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
