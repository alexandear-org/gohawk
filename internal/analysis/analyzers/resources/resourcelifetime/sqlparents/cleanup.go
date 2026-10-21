// Package sqlparents proves exact SQL parent cleanup relationships. Closing a
// database or finishing a transaction supplies parent-lifetime evidence; callers
// decide whether that evidence settles a child resource or leaves it uncertain.
package sqlparents

import (
	"github.com/kojah/gohawk/internal/engine/heapmodel"
	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// Kind names the parent contract established by an exact identity proof.
type Kind uint8

const (
	// Unrelated means no parent relationship was established.
	Unrelated Kind = iota
	// DatabaseClosed identifies DB.Close for a statement prepared on that DB.
	DatabaseClosed
	// TransactionFinished identifies Commit/Rollback for rows from that Tx.
	TransactionFinished
)

// Proof retains cutoff separately from a completed identity disproof. Kind is
// meaningful for positive evidence; a cutoff never establishes a parent.
type Proof struct {
	State proofs.EvidenceState
	Kind  Kind
}

func result(found bool, kind Kind, budget *proofs.SearchBudget) Proof {
	if exhausted(budget) {
		return Proof{State: proofs.EvidenceUnknown}
	}
	if found {
		return Proof{State: proofs.EvidenceProven, Kind: kind}
	}
	return Proof{State: proofs.EvidenceDisproven}
}

func exhausted(budget *proofs.SearchBudget) bool { return budget.Exhausted() || budget.PoolExhausted() }

func databaseCall(common *ssa.CallCommon, names ...string) bool {
	return receiverCall(common, "DB", names...)
}

func receiverCall(common *ssa.CallCommon, receiver string, names ...string) bool {
	for _, name := range names {
		if ssacall.CallMatchesSymbol(common, syntax.PackageMethod(syntax.MethodSymbol{PackagePath: "database/sql", Receiver: receiver, Name: name})) {
			return true
		}
	}
	return false
}

// ProveWithin checks only applicable standard SQL calls before spending an
// allowance. Storage children retain their own cap and share the caller pool.
//
// DB.Close retires the connections that own DB-prepared driver statements.
// This is not a claim that the Stmt becomes unusable immediately: outstanding
// connection users can delay release. Rows, transactions, and Conn-prepared
// statements keep their own obligations. Require the captured receiver itself,
// not an existential alias through a mutable cell or a mixed-parent phi.
// https://github.com/mariadb-operator/mariadb-operator/blob/e8ece7a8076954674e10e0381571bd80278ac35f/licenses/go-licenses/github.com/go-sql-driver/mysql/benchmark_test.go#L377-L407
// Finishing a transaction cancels the transaction context watched by active
// Rows, including rows from statements prepared on that exact transaction.
// The cancellation may close rows asynchronously, so this establishes an
// opaque parent-owned lifetime, not a synchronous Rows.Close guarantee.
// DB.Close and Stmt.Close do not have this contract for their active rows.
// https://github.com/bluesky-social/indigo/blob/41278964ec8e3253e70d4e919dfb8e34211c543d/carstore/sqlite_store.go#L171-L190
func ProveWithin(acquisition *ssa.Call, instruction ssa.Instruction, budget *proofs.SearchBudget) Proof {
	// Symbol applicability precedes any query. An unrelated helper must reach
	// its own completion proof and exhaustion event rather than a SQL cutoff.
	unmatched := Proof{State: proofs.EvidenceDisproven, Kind: Unrelated}
	if acquisition == nil {
		return unmatched
	}
	switch instruction.(type) {
	case *ssa.Call, *ssa.Defer:
	default:
		return unmatched
	}
	common := ssaflow.InstructionCall(instruction)
	databaseParent := databaseCall(acquisition.Common(), "Prepare", "PrepareContext") && databaseCall(common, "Close")
	transactionParent := receiverCall(common, "Tx", "Commit", "Rollback") &&
		(receiverCall(acquisition.Common(), "Tx", "Query", "QueryContext") ||
			receiverCall(acquisition.Common(), "Stmt", "Query", "QueryContext"))
	if !databaseParent && !transactionParent {
		return unmatched
	}
	if !budget.Spend() {
		return result(false, Unrelated, budget)
	}
	if databaseParent {
		return parentIdentityWithin(ssaflow.CallReceiver(common), ssaflow.CallReceiver(acquisition.Common()),
			DatabaseClosed, budget)
	}
	parent := rowsTransactionWithin(acquisition, budget)
	if parent.State != proofs.EvidenceProven {
		return parent.Proof
	}
	return parentIdentityWithin(ssaflow.CallReceiver(common), parent.Parent, TransactionFinished, budget)
}

type rowsTransactionProof struct {
	Proof
	Parent ssa.Value
}

func rowsTransactionWithin(acquisition *ssa.Call, budget *proofs.SearchBudget) rowsTransactionProof {
	if !budget.Spend() {
		return rowsTransactionProof{Proof: result(false, Unrelated, budget)}
	}
	common := acquisition.Common()
	if receiverCall(common, "Tx", "Query", "QueryContext") {
		return rowsTransactionProof{
			Proof:  result(true, TransactionFinished, budget),
			Parent: ssaflow.CallReceiver(common),
		}
	}
	if !receiverCall(common, "Stmt", "Query", "QueryContext") {
		return rowsTransactionProof{Proof: result(false, Unrelated, budget)}
	}
	// Require the exact constructor result. A replaced statement, mixed phi or
	// constructor on another parent must retain its independent obligation.
	child := budget.Within(proofs.QueryBudget)
	statement := heapmodel.NewStorage(child).Resolve(ssaflow.CallReceiver(common))
	if exhausted(child) {
		return rowsTransactionProof{Proof: result(false, Unrelated, child)}
	}
	extract, ok := statement.Value.(*ssa.Extract)
	if !statement.Proven() || !ok || extract.Index != 0 {
		return rowsTransactionProof{Proof: result(false, Unrelated, budget)}
	}
	prepare, ok := extract.Tuple.(*ssa.Call)
	if !ok || !receiverCall(prepare.Common(), "Tx", "Prepare", "PrepareContext") {
		return rowsTransactionProof{Proof: result(false, Unrelated, budget)}
	}
	return rowsTransactionProof{
		Proof:  result(true, TransactionFinished, budget),
		Parent: ssaflow.CallReceiver(prepare.Common()),
	}
}

// Later closure captures can spill a local into a cell. Two loads agree only
// when their point-in-time stored values agree; a shared address alone must
// never accept a reassigned DB. Graph and type internals retain separate costs.
func parentIdentityWithin(left, right ssa.Value, reason Kind, budget *proofs.SearchBudget) Proof {
	child := budget.Within(proofs.QueryBudget)
	proof := heapmodel.NewStorage(child).Same(left, right)
	return result(proof.Proven(), reason, child)
}
