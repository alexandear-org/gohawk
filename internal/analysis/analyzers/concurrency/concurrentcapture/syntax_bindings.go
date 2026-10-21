package concurrentcapture

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// Syntax recognition supplies one root step or semaphore operation. Traversal,
// object identity and the lifetime of a pending send stay with the caller.
type (
	rootStepSignature         func(*analysis.Pass, ast.Expr) (*ast.Ident, ast.Expr)
	channelStatementSignature func(ast.Stmt) (channelStatementKind, ast.Expr)
)

var (
	_ rootStepSignature         = mutationRootStep
	_ channelStatementSignature = channelStatement
)

type channelStatementKind uint8

const (
	channelStatementNone channelStatementKind = iota
	channelStatementSend
	channelStatementReceive
)
const channelReceiveToken = token.ARROW

func mapSource(pass *analysis.Pass, expression ast.Expr) bool {
	typ := pass.TypesInfo.TypeOf(expression)
	if typ == nil {
		return false
	}
	_, ok := typ.Underlying().(*types.Map)
	return ok
}
