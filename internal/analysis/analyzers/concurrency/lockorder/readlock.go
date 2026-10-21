package lockorder

import (
	"fmt"
	"go/types"
	"slices"

	"github.com/kojah/gohawk/internal/analysis/analyzers/concurrency/lockorder/lockforms"
	"github.com/kojah/gohawk/internal/analysis/analyzers/concurrency/lockorder/lockpolicy"
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"github.com/kojah/gohawk/internal/reporting/check"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ssa"
)

// A read lock is shared: sync.RWMutex documents RLock as held by any number of
// readers at once. Writing to the object that lock protects while holding only
// RLock therefore races with every other reader, and unlike an order inversion
// it needs nothing unusual from a second goroutine -- two ordinary readers are
// enough.
//
// The claim is deliberately about the OBJECT, not about which field the lock
// guards. Proving that a mutex guards a particular field needs guard inference
// this analyzer does not do, and one struct may hold several independent guard
// domains: a mutex over one map, an atomic counter, and fields fixed at
// construction. What is provable without that inference is narrower: the
// receiver whose read lock is held is the receiver being mutated.
//
// A field also written in a call-free receiver method has an opaque caller
// synchronization contract and stays unknown; see readLockFieldEvidence.
// Other exclusions follow from the same rule and keep the claim honest. A value
// LOADED out of the owner is a different cell, so mutating it is not a write to
// the owner -- the distinction ssaflow.IdentitySource states for identity
// resolution. And an atomic update is a call rather than a store, so it never
// reaches here at all, which is correct: such a field is protected by atomics
// rather than by the lock.

// reportReadLockWrites reports a write to an object whose read lock is the only
// one held at this instruction.
func reportReadLockWrites(
	flow lockFlowContext, instruction ssa.Instruction, held, readHeld []string,
	lockValues map[string][]ssa.Value, possibleWriters []*ssa.Defer,
) {
	proof := proveReadLockWrite(instruction, held, readHeld, lockValues, possibleWriters, flow.setup.calls, flow.fieldEvidence, flow.budget)
	traceLockDiagnostic(flow.pass, check.LockReadLockWrite, instruction.Pos(), proof.lockDiagnosticProof)
	if proof.state != proofs.EvidenceProven {
		return
	}
	// Branch evidence can visit one write repeatedly. Keep proof and tracing
	// per state, but publish each instruction/lock witness only once. Unknown
	// states never reserve a witness, and a different held lock stays distinct.
	// https://github.com/apache/skywalking-rover/blob/e83d5925500a7e63dd55c080a9b1542d6cedaefb/pkg/tools/buffer/buffer.go#L644-L649
	witness := readLockWriteWitness{instruction: instruction, identity: proof.identity}
	if flow.readLockWrites[witness] {
		return
	}
	flow.readLockWrites[witness] = true
	source := syntax.SourceRange(flow.pass, instruction.Pos())
	check.Report(flow.pass, check.LockReadLockWrite, analysis.Diagnostic{
		Pos: source.Pos(), End: source.End(),
		Message: fmt.Sprintf("write while only the read lock %s is held", flow.lockName(proof.identity)),
		Related: flow.acquisitionEvidence(proof.identity),
	})
}

type readLockWriteWitness struct {
	instruction ssa.Instruction
	identity    string
}

type readLockWriteProof struct {
	lockDiagnosticProof
	identity string
}

// proveReadLockWrite selects the first reportable owner under current lock
// state. An exclusive or possible imported writer leaves this mutation unknown;
// neither is a field-protection contract. Unknown candidates retain the existing
// full owner-query order, while a reportable candidate ends the search.
func proveReadLockWrite(
	instruction ssa.Instruction, held, readHeld []string,
	lockValues map[string][]ssa.Value, possibleWriters []*ssa.Defer, calls []*ssa.Call, fields readLockFieldEvidence, budget *proofs.SearchBudget,
) readLockWriteProof {
	proof := readLockWriteProof{lockDiagnosticProof: lockDiagnosticProof{proofs.EvidenceDisproven, lockReasonNone}}
	for _, identity := range readHeld {
		// A lock the flow already transferred or released is no longer held,
		// even if it was taken for reading earlier on this path.
		if !slices.Contains(held, identity) {
			continue
		}
		for _, value := range lockValues[identity] {
			owner, ok := lockOwner(value)
			if !ok || !writeTargetsOwner(instruction, owner) {
				continue
			}
			step := lockpolicy.ReadWrite(readWriteQueries{
				instruction: instruction, held: held, readHeld: readHeld,
				possibleWriters: possibleWriters, calls: calls, fields: fields, budget: budget,
			})
			proof = readLockWriteProof{lockDiagnosticProof: lockDiagnosticProof{state: step.Proof.State, reason: step.Proof.Reason}, identity: identity}
			if step.Stop {
				return proof
			}
		}
	}
	return proof
}

// writeLockHeld reports whether this path also holds an exclusive lock.
func writeLockHeld(held, readHeld []string) bool {
	return slices.ContainsFunc(held, func(identity string) bool {
		return !slices.Contains(readHeld, identity)
	})
}

// lockOwner returns the value a lock is a field of. A package variable has no
// owner object, so what it protects is not decided here.
func lockOwner(value ssa.Value) (ssa.Value, bool) { //nolint:ireturn // SSA values have several concrete forms.
	field, ok := value.(*ssa.FieldAddr)
	if !ok {
		return nil, false
	}
	return field.X, true
}

func writeTargetsOwner(instruction ssa.Instruction, owner ssa.Value) bool {
	destination, direct := lockforms.Destination(instruction)
	if destination == nil {
		return false
	}
	if direct {
		return addressWithinOwner(destination, owner)
	}
	// Containers loaded out of the owner's field retain their storage path.
	source, ok := ssaflow.IdentitySource(destination)
	return ok && addressWithinOwner(source, owner)
}

// readLockWriteDestination names the memory a store, map update or supported
// builtin mutates. Owner matching and exclusivity consume this same selection.
// copy writes only its first argument; its source remains a permitted read.
func readLockWriteDestination(instruction ssa.Instruction) ssa.Value { //nolint:ireturn // Preserve the exact SSA storage identity.
	destination, _ := lockforms.Destination(instruction)
	return destination
}

// addressWithinOwner reports whether address selects a field or a constant
// index of owner. It never peels a load, so a value read out of the owner and
// then mutated is not counted as a write to the owner.
func addressWithinOwner(address, owner ssa.Value) bool {
	for address != nil {
		if heapmodel.MayAlias(address, owner) {
			return true
		}
		next, known := lockforms.OwnerStep(address)
		if !known {
			return false
		}
		address = next
	}
	return false
}

// Receiver-field writes in a call-free method have no in-frame acquisition.
// Whether callers provide a guard or own the field is opaque. Declaration
// identity keeps this uncertainty on that field, not on its siblings or names.
// Interrupted collection supplies no negative field-guard evidence.
type readLockFieldEvidence struct {
	unscoped    map[*types.Var]bool
	unavailable bool
}

func collectReadLockFieldEvidence(functions []*ssa.Function, budget *proofs.SearchBudget) readLockFieldEvidence {
	result := readLockFieldEvidence{unscoped: map[*types.Var]bool{}}
	for _, function := range functions {
		if !budget.Spend() {
			return readLockFieldEvidence{unavailable: true}
		}
		if function.Signature.Recv() == nil || len(function.Params) == 0 || len(function.Blocks) == 0 {
			continue
		}
		// Stage witnesses until the entire method is known call-free. A later
		// call can provide synchronization that a preceding store does not show.
		var fields []*types.Var
		closed := true
		for instruction := range ssaflow.InstructionsWithin(function, budget) {
			if _, call := instruction.(ssa.CallInstruction); call {
				closed = false
				break
			}
			store, ok := instruction.(*ssa.Store)
			if !ok {
				continue
			}
			// A field of a loaded peer is a different object. Only the exact
			// receiver and its embedded addresses supply this declaration witness.
			path, known := ssaflow.ResolveEmbeddedFieldPath(ssaflow.NewReachingWalk(ssaflow.TransparentNone).Within(budget),
				store.Addr, func(root ssa.Value) bool { return root == function.Params[0] })
			if known && path.Depth > 0 {
				if field := storedField(store); field != nil {
					fields = append(fields, field)
				}
			}
		}
		if budget.Exhausted() || budget.PoolExhausted() {
			return readLockFieldEvidence{unavailable: true}
		}
		if closed {
			for _, field := range fields {
				if !budget.Spend() {
					return readLockFieldEvidence{unavailable: true}
				}
				result.unscoped[field] = true
			}
		}
	}
	return result
}

func (fields readLockFieldEvidence) guard(instruction ssa.Instruction) lockDiagnosticProof {
	if fields.unavailable {
		return lockDiagnosticProof{proofs.EvidenceUnknown, lockReasonLockStateBudgetExhausted}
	}
	if store, ok := instruction.(*ssa.Store); ok && fields.unscoped[storedField(store)] {
		return lockDiagnosticProof{proofs.EvidenceUnknown, lockReasonFieldGuardUnknown}
	}
	return lockDiagnosticProof{proofs.EvidenceDisproven, lockReasonNone}
}

func storedField(store *ssa.Store) *types.Var {
	address, ok := store.Addr.(*ssa.FieldAddr)
	if !ok {
		return nil
	}
	structure := syntax.PointerStruct(address.X.Type())
	if structure == nil || address.Field < 0 || address.Field >= structure.NumFields() {
		return nil
	}
	return structure.Field(address.Field)
}

// One matched owner is decided at a time. The caller retains candidate order,
// accumulated fallback evidence, identity binding and the stop/continue action.
type readWriteQueries struct {
	instruction     ssa.Instruction
	held, readHeld  []string
	possibleWriters []*ssa.Defer
	calls           []*ssa.Call
	fields          readLockFieldEvidence
	budget          *proofs.SearchBudget
}

func (q readWriteQueries) Guard() lockpolicy.Proof {
	guard := q.fields.guard(q.instruction)
	return lockpolicy.Proof{State: guard.state, Reason: guard.reason}
}

func (q readWriteQueries) PrivateStorage() bool {
	storage, known := heapmodel.ExclusiveAt(readLockWriteDestination(q.instruction), q.instruction)
	return known && storage.Local
}
func (q readWriteQueries) HeldWriter() bool { return writeLockHeld(q.held, q.readHeld) }
func (q readWriteQueries) Writer() lockpolicy.WriterEvidence {
	return provePossibleWriterAt(q.possibleWriters, q.instruction, q.calls, q.budget)
}
