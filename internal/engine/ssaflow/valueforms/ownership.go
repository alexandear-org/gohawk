package valueforms

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// OwnershipSignature exposes one syntactic holder candidate. The caller owns
// traversal and decides whether the candidate establishes external ownership.
type OwnershipSignature func(ssa.Value) (ssa.Value, bool)

var _ OwnershipSignature = OwnershipSource

const (
	loadToken    = token.MUL
	receiveToken = token.ARROW
)
