package goroutineownership

import (
	"testing"

	proofs "github.com/kojah/gohawk/internal/engine/proof"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/ssa"
)

func TestAggregateRootFollowsOnlyAggregateOperands(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "aggregatebinding", `package aggregatebinding
type signal chan int
type item struct{ done chan int }
type owner struct{ items []item }
func nested(p *owner, index int) chan int { return p.items[index].done }
func field(p item) chan int { return p.done }
func indexed(items []chan int, index int) chan int { return items[index] }
func lookup(items map[int]chan int, key int) chan int { return items[key] }
func wrapped(p *owner, index int) any { return p.items[index].done }
func sliced(items []int) []int { return items[1:] }
func received(channel chan signal) signal { return <-channel }
`)
	for _, name := range []string{"nested", "field", "indexed", "lookup", "wrapped", "sliced", "received"} {
		t.Run(name, func(t *testing.T) {
			function := pkg.Func(name)
			value := ssaflow.InstructionsOf[*ssa.Return](function)[0].Results[0]
			var want ssa.Value = function.Params[0]
			if name == "field" {
				// A by-value struct parameter is copied into a frame-local cell.
				// Projection must stop at that copy rather than invent identity
				// with the original parameter.
				want = ssaflow.InstructionsOf[*ssa.Alloc](function)[0]
			}
			if name == "sliced" || name == "received" {
				want = value
			}
			if got := aggregateRoot(value); got != want {
				t.Fatalf("root=%v, want %v", got, want)
			}
			budget := proofs.NewSearchBudget(0)
			if got := aggregateRootWithin(value, budget); got != nil || !budget.Exhausted() {
				t.Fatalf("cutoff root=%v, exhausted=%t", got, budget.Exhausted())
			}
		})
	}
}
