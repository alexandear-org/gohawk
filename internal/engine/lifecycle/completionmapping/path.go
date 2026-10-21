package completionmapping

import (
	"github.com/kojah/gohawk/internal/engine/ssaflow"
)

// RequiresNestedPath excludes completion of an owner or containing aggregate
// unless the nested answer names the caller target's exact field.
func (local Local) RequiresNestedPath() bool {
	return local.Kind == Owner || local.Kind == Exact && len(local.Path) > 0
}

func appendPath(supplied, actual []string) string {
	return ssaflow.JoinAccessPath(append(append([]string(nil), supplied...), actual...))
}
