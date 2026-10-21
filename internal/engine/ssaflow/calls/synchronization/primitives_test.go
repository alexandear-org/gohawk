package synchronization_test

import (
	"testing"

	"github.com/kojah/gohawk/internal/engine/ssaflow/calls/synchronization"
	"github.com/kojah/gohawk/internal/engine/ssaflow/ssaflowtest"
	"golang.org/x/tools/go/ssa"
)

func TestStandardPrimitiveContracts(t *testing.T) {
	pkg := ssaflowtest.BuildPackage(t, "example.com/primitives", `package primitives
 import "sync"
 type Lookalike struct{}
 func(*Lookalike) Lock(){}
 func(*Lookalike) Wait(){}
 func Model(mu *sync.Mutex,rw *sync.RWMutex,wg *sync.WaitGroup,c *sync.Cond,ch chan int,n int,fake *Lookalike){
  mu.Lock();mu.Unlock();rw.Lock();rw.Unlock();rw.RLock();rw.RUnlock()
  c.Wait();close(ch);wg.Done();wg.Wait();wg.Add(1);wg.Add(2);wg.Add(n)
  fake.Lock();fake.Wait();mu.TryLock();rw.TryLock();rw.TryRLock()
 }
 `)
	expected := []struct {
		kind              synchronization.Kind
		valid, recognized bool
	}{
		{synchronization.Lock, true, true},
		{synchronization.Unlock, true, true},
		{synchronization.Lock, true, true},
		{synchronization.Unlock, true, true},
		{synchronization.ReadLock, true, true},
		{synchronization.ReadUnlock, true, true},
		{synchronization.CondWait, true, true},
		{synchronization.Close, true, true},
		{synchronization.GroupDone, true, true},
		{synchronization.GroupWait, true, true},
		{synchronization.GroupAdd, true, true},
		{synchronization.GroupAdd, false, true},
		{synchronization.GroupAdd, false, true},
		{0, false, false},
		{0, false, false},
		{0, false, false},
		{0, false, false},
		{0, false, false},
	}
	index := 0
	for _, block := range pkg.Func("Model").Blocks {
		for _, instruction := range block.Instrs {
			call, ok := instruction.(*ssa.Call)
			if !ok {
				continue
			}
			if index >= len(expected) {
				t.Fatal("unexpected call", call)
			}
			want := expected[index]
			got, recognized := synchronization.Recognize(call.Common())
			valid := got.Kind != synchronization.GroupAdd || got.Fixed && got.Count == 1
			if recognized != want.recognized || recognized && (got.Kind != want.kind || valid != want.valid || got.Resource != call.Common().Args[0]) {
				t.Fatalf("call %d %s: got %+v,%v want %+v", index, call, got, recognized, want)
			}
			_, mutation := synchronization.MutexEffect(call.Common())
			if wantMutation := index < 6 || index >= 15; mutation != wantMutation {
				t.Fatalf("call %d: mutex storage effect=%v, want %v", index, mutation, wantMutation)
			}
			index++
		}
	}
	if index != len(expected) {
		t.Fatalf("checked %d calls, want %d", index, len(expected))
	}
	if _, recognized := synchronization.Recognize(nil); recognized {
		t.Fatal("nil call recognized")
	}
}
