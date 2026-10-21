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
| Call-use and heap-operation classification | Handwritten; migration remains; state application stays in Go |
| Lifecycle completion/transfer recognition and outcome selection | Handwritten; migration remains; recursive searches stay in Go |
| Goroutine ownership instruction/call labels | Handwritten; migration remains with ordered proof queries |
| Cancellation ownership instruction/call labels | Generated instruction, direct/deferred, call-contract, exact-argument and return decisions; owner discovery, searches, cache and flow stay in Go |
| Process ownership decision selection | Generated post-Start return policy with ordered command-use and entry-lifetime queries; instruction classification remains |
| Resource lifetime classification and acquisition-boundary patterns | Handwritten; migration remains; catalog collection stays a table |
| Defer-in-loop resource classification | Generated per-value and per-acquisition decisions; call discovery and loop traversal remain in Go |
| Lock operation, write-target, and held-state transition families | Handwritten; migration remains; identity and cycle search stay in Go |
| Concurrent capture syntax and guard decisions | Handwritten; migration remains for substantial pattern/decision families |
| Producer lifecycle use classification | Handwritten; migration remains for instruction classifications |
| Result and lifecycle summary leaf recognition | Handwritten; migration remains; summary inference stays in Go |

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
