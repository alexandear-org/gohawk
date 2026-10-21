package admission

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

type (
	// DataSignature decides one instruction's inert-data admission. Later uses
	// remain independently classified by the summary collector.
	DataSignature func(ssa.Instruction) bool
	node          = ssa.Value
)

var _ DataSignature = DataInstruction

const (
	equalToken    = token.EQL
	notEqualToken = token.NEQ
)
