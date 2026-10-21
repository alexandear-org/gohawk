package cancellationownership

import (
	"slices"

	"github.com/kojah/gohawk/internal/engine/heapmodel"
	"github.com/kojah/gohawk/internal/engine/ssaflow"
	ssacall "github.com/kojah/gohawk/internal/engine/ssaflow/calls"
	ssapath "github.com/kojah/gohawk/internal/engine/ssaflow/path"
	"github.com/kojah/gohawk/internal/engine/syntax"
	"golang.org/x/tools/go/ssa"
)

// Classification labels each instruction once for the cancellation flow.
// Exact invocation and transfer stay distinct from opaque consumption; parent
// context and Done observations supply uncertainty rather than exact cleanup.

func (classifier *cancellationClassifier) action(instruction ssa.Instruction) cancellationAction {
	if action, ok := classifier.actions[instruction]; ok {
		return action
	}
	label := classifier.classifyAction(instruction)
	action, reason := label.action, label.reason
	if action == cancellationActionNone && classifier.parent != nil && classifier.parent.action(instruction) != cancellationActionNone {
		action, reason = cancellationActionUnknown, reasonLabelParentContextUse
	}
	if returned, ok := instruction.(*ssa.Return); ok {
		// Flow can revisit a merged return under different path states. Its
		// result and owner evidence is instruction-local: combine it with the
		// ordinary label once, so neither queries nor trace labels repeat.
		// Exact return cleanup/transfer still covers an opaque ordinary use.
		returnedLabel := classifier.returnLabel(returned)
		if returnedLabel.action != cancellationActionNone && cancellationObligation(returnedLabel.action) >= cancellationObligation(action) {
			action, reason = returnedLabel.action, returnedLabel.reason
		}
	}
	classifier.actions[instruction] = action
	classifier.traceLabel(instruction, action, reason)
	if action == cancellationActionTransfer {
		classifier.transfers = true
	}
	return action
}

func (classifier *cancellationClassifier) classifyAction(instruction ssa.Instruction) cancellationLabel {
	return classifyCancellation(instruction, classifier)
}

func (classifier *cancellationClassifier) recognizedAction(
	instruction ssa.Instruction,
	common *ssa.CallCommon,
) (cancellationLabel, bool) {
	return recognizeCancellation(instruction, classifier, common)
}

func (classifier *cancellationClassifier) recognizedDirectAction(
	instruction ssa.Instruction,
	common *ssa.CallCommon,
) (cancellationLabel, bool) {
	return recognizeCancellationDirect(instruction, classifier, common)
}

func (classifier *cancellationClassifier) recognizedCallAction(
	instruction ssa.Instruction,
	common *ssa.CallCommon,
) (cancellationLabel, bool) {
	return recognizeCancellationCall(instruction, classifier, common)
}

// exactArgumentAction labels a call that passes the exact cancel function
// as an argument: launched, proven called, or undecided.
func (classifier *cancellationClassifier) exactArgumentAction(instruction ssa.Instruction) (cancellationLabel, bool) {
	return recognizeCancellationArgument(instruction, classifier)
}

func deferredClosureCaptures(instruction ssa.Instruction, target ssa.Value) bool {
	if _, ok := instruction.(*ssa.Defer); !ok {
		return false
	}
	common := ssaflow.InstructionCall(instruction)
	if common == nil {
		return false
	}
	closure, ok := common.Value.(*ssa.MakeClosure)
	return ok && slices.ContainsFunc(closure.Bindings, func(binding ssa.Value) bool {
		return heapmodel.CapturedBindingMatches(binding, target)
	})
}

func (classifier *cancellationClassifier) returnLabel(returned *ssa.Return) cancellationLabel {
	return classifyCancellationReturn(returned, classifier)
}

// Receiving this standard context's Done observes cancellation, not just an
// intention to cancel. Keep it unknown rather than claiming synchronous parent
// unlinking. NotifyContext is excluded: receiving a signal does not unregister
// its handler. Each select arm must be proved separately; another case or a
// default arm must not borrow this evidence.
// https://github.com/chrislusf/gleam/blob/8b4ae277059f30322db71de5e0473a4351bf9f8d/util/context.go#L7-L25
func (classifier *cancellationClassifier) ownDoneChannel(value ssa.Value) bool {
	call, ok := value.(*ssa.Call)
	return ok && classifier.context != nil && ssaflow.CallReceiver(call.Common()) == classifier.context &&
		ssacall.CallMatchesSymbol(call.Common(), syntax.PackageMethod(syntax.MethodSymbol{
			PackagePath: "context", Receiver: "Context", Name: "Done",
		}))
}

func (classifier *cancellationClassifier) selectedDoneEdge(from, to *ssa.BasicBlock) bool {
	if classifier.context == nil {
		return false
	}
	channel, selected := ssapath.SelectedReceiveOnEdge(from, to)
	return selected && classifier.ownDoneChannel(channel)
}

func commonHasExactArgument(common *ssa.CallCommon, target ssa.Value) bool {
	return slices.Contains(common.Args, target)
}
