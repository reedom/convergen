package model

import (
	"go/types"

	"github.com/reedom/convergen/pkg/util"
)

func IterateStructFields(structNode Node, cb func(Node) (done bool)) {
	util.IterateFields(structNode.ExprType(), func(t *types.Var) (done bool) {
		node := NewStructFieldNode(structNode, t)
		return cb(node)
	})
}
func IterateStructMethods(structNode Node, cb func(Node) (done bool)) {
	util.IterateMethods(structNode.ExprType(), func(fn *types.Func) (done bool) {
		if !util.CompliesGetter(fn) {
			return
		}
		node := NewStructMethodNode(structNode, fn)
		return cb(node)
	})
}

// IsRecursive reports whether it'd form a recursive structure if it added the newChild.
func IsRecursive(node Node, typ types.Type) bool {
	childType := util.DerefPtr(typ)
	if !util.IsNamedType(childType) {
		return false
	}

	childStr := childType.String()
	for p := node; p != nil; p = p.Parent() {
		if p.ExprType().String() == childStr {
			return true
		}
	}
	return false
}
