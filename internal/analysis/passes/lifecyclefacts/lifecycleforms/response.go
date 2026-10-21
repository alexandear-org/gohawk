// Package lifecycleforms identifies structural leaves for lifecycle summaries.
// It preserves exact SSA nodes; acquisition, freshness and release claims still
// require the enclosing pass's identity, flow and imported-summary evidence.
package lifecycleforms

import (
	"go/token"

	"github.com/kojah/gohawk/internal/engine/syntax"

	"golang.org/x/tools/go/ssa"
)

type responseSignature func(ssa.Value) *ssa.FieldAddr

var _ responseSignature = ResponseBody

const loadToken = token.MUL

func responseBodyField(field *ssa.FieldAddr) bool {
	if !syntax.NamedType(field.X.Type(), "net/http", "Response") {
		return false
	}
	structure := syntax.PointerStruct(field.X.Type())
	return structure != nil && structure.Field(field.Field).Name() == "Body"
}
