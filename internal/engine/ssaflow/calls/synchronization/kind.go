// Package synchronization recognizes synchronization operations without resolving
// resource identity, control flow, or the completeness of a summary.
package synchronization

import "strconv"

// Kind identifies a synchronization event with an exact resource.
type Kind uint8

const (
	Send Kind = iota
	Receive
	Close
	GroupAdd
	GroupDone
	GroupWait
	Lock
	Unlock
	CondWait
	// Cancel requests cancellation; it neither joins a worker nor proves that
	// Done has closed before the call returns.
	Cancel
	ReadLock
	ReadUnlock
	// Invoke calls a function-typed input, or a method of an interface input,
	// at this point. It is a hole, not an effect: binding replaces it with the
	// supplied function's or concrete method's effects, and a summary that
	// still contains one is incomplete until callback binding resolves it.
	Invoke
)

// String names the synchronization event.
func (kind Kind) String() string {
	names := [...]string{
		Send: "send on", Receive: "receive from", Close: "close", GroupAdd: "WaitGroup.Add on", GroupDone: "WaitGroup.Done on",
		GroupWait: "WaitGroup.Wait on", Lock: "Lock", Unlock: "Unlock", CondWait: "Cond.Wait on", Cancel: "cancel",
		ReadLock: "RLock", ReadUnlock: "RUnlock", Invoke: "invoke",
	}
	if int(kind) < len(names) {
		return names[kind]
	}
	return "effect " + strconv.Itoa(int(kind))
}
