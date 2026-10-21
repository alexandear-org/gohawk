// Package resourcepolicy owns ordered resource-consumption decisions and the
// reason vocabulary shared with the analyzer. Reached queries retain their
// own searches, allowances, caches and observation points. Unknown evidence
// is terminal and cannot establish transparent consumption.
package resourcepolicy

import (
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"golang.org/x/tools/go/ssa"
)

// Evidence retains analyzer-owned proof payloads across policy decisions.
type Evidence struct {
	State      proofs.EvidenceState
	Reason     Reason
	Provenance proofs.EvidenceProvenance
}

// Storage retains the resolved destination owner without repeating its query.
type Storage struct {
	State  proofs.EvidenceState
	Reason Reason
	Owner  ssa.Value
}

// InstructionQueries supplies consumption evidence for one SSA instruction.
type InstructionQueries interface {
	Returned(*ssa.Return) Evidence
	Storage(*ssa.Store) Storage
	Wrapper(*ssa.Store) Evidence
	Handoff(ssa.Value, ssa.Instruction) Evidence
	Payload(ssa.Value, Reason) (Reason, bool)
	Select(*ssa.Select) (Reason, bool)
	Call(ssa.Instruction, *ssa.CallCommon) (Reason, bool)
}

// CallQueries supplies the ordered boundaries before named-function evidence.
type CallQueries interface {
	Carried(*ssa.CallCommon) Evidence
	Callback(ssa.Instruction, *ssa.CallCommon) Evidence
	Exposure(ssa.Instruction, *ssa.CallCommon) Evidence
	Closure(ssa.Instruction, *ssa.MakeClosure, bool) (Reason, bool)
	Function(ssa.Instruction, *ssa.CallCommon, bool) (Reason, bool)
}

// FunctionQueries supplies retention and result-publication evidence.
type FunctionQueries interface {
	Chain(ssa.Instruction, *ssa.CallCommon) Evidence
	Escape(ssa.Instruction, *ssa.CallCommon) Evidence
	Aggregate(*ssa.CallCommon) Evidence
	Publication(ssa.Instruction) Evidence
	Unmodeled(ssa.Instruction, *ssa.CallCommon) bool
}

type (
	storeSignature         func(InstructionQueries, *ssa.Store) (Reason, bool)
	consumeSignature       func(InstructionQueries, ssa.Instruction) (Reason, bool)
	sendSignature          func(InstructionQueries, ssa.Value, ssa.Instruction) (Reason, bool)
	callSignature          func(CallQueries, ssa.Instruction, *ssa.CallCommon) (Reason, bool)
	afterCallbackSignature func(CallQueries, ssa.Instruction, *ssa.CallCommon, bool) (Reason, bool)
	dispatchSignature      func(CallQueries, ssa.Instruction, *ssa.CallCommon, bool) (Reason, bool)
	functionSignature      func(FunctionQueries, ssa.Instruction, *ssa.CallCommon, bool) (Reason, bool)
	functionTailSignature  func(FunctionQueries, ssa.Instruction, *ssa.CallCommon, bool) (Reason, bool)
)

var (
	_ storeSignature         = StoreConsumption
	_ consumeSignature       = Consume
	_ sendSignature          = Send
	_ callSignature          = Call
	_ afterCallbackSignature = AfterCallback
	_ dispatchSignature      = Dispatch
	_ functionSignature      = Function
	_ functionTailSignature  = FunctionTail
)

const (
	evidenceProven    = proofs.EvidenceProven
	evidenceDisproven = proofs.EvidenceDisproven
	evidenceUnknown   = proofs.EvidenceUnknown
)

func collectionOwner(store *ssa.Store, evidence Storage) bool {
	_, field := store.Addr.(*ssa.FieldAddr)
	return field && evidence.Owner != nil && ssaflow.ElementOfAggregate(evidence.Owner)
}
func invokes(common *ssa.CallCommon) bool   { return common.IsInvoke() }
func appendBuiltin(value *ssa.Builtin) bool { return value.Name() == "append" }
func dynamic(common *ssa.CallCommon) bool   { return common.StaticCallee() == nil }

// Unknown distinguishes an interrupted or unavailable proof from known absence.
func (e Evidence) Unknown() bool { return e.State == proofs.EvidenceUnknown }

// Proven reports positive evidence without discarding its reason payload.
func (e Evidence) Proven() bool { return e.State == proofs.EvidenceProven }

// Unknown preserves storage uncertainty before inspecting its resolved owner.
func (e Storage) Unknown() bool { return e.State == proofs.EvidenceUnknown }

func possible(e Evidence) bool { return e.State != proofs.EvidenceDisproven }
