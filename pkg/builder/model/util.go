package model

import (
	"go/types"

	"github.com/reedom/convergen/v8/pkg/util"
)

// IterateStructFields iterates through all the fields of the given structNode and
// calls the given callback function for each one.
// If the callback function returns true, the iteration stops.
func IterateStructFields(structNode Node, cb func(Node) (done bool)) {
	util.IterateFields(structNode.ExprType(), func(t *types.Var) (done bool) {
		node := NewStructFieldNode(structNode, t)
		return cb(node)
	})
}

// IterateStructMethods iterates through all the methods of the given structNode and
// calls the given callback function for each one.
// Only those methods that comply with the getter compliant method will be processed.
// If the callback function returns true, the iteration stops.
func IterateStructMethods(structNode Node, cb func(Node) (done bool)) {
	util.IterateMethods(structNode.ExprType(), func(fn *types.Func) (done bool) {
		if !util.CompliesGetter(fn) {
			return
		}
		node := NewStructMethodNode(structNode, fn)
		return cb(node)
	})
}

// IsRecursive checks if the given type is the same as any of the ancestor nodes in the
// tree rooted at the given node.
// If true, it means there is a recursive reference in the tree.
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
