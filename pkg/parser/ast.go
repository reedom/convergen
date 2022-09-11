package parser

import (
	"go/ast"
	"go/types"
	"regexp"

	"golang.org/x/tools/go/ast/astutil"
)

// astToNodes converts types.Object to []ast.Node.
func astToNodes(file *ast.File, obj types.Object) (path []ast.Node, exact bool) {
	return astutil.PathEnclosingInterval(file, obj.Pos(), obj.Pos())
}

// astGetDocComments retrieves doc comments that relate to nodes.
func astGetDocCommentOn(file *ast.File, obj types.Object) *ast.CommentGroup {
	nodes, _ := astToNodes(file, obj)
	if nodes == nil {
		return nil
	}

	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.GenDecl:
			return n.Doc
		case *ast.FuncDecl:
			return n.Doc
		default:
			continue
		}
	}
	return nil
}

// astRemoveMatchComments removes pattern patched comments from file.Comments.
func astRemoveMatchComments(file *ast.File, pattern *regexp.Regexp) {
	for _, group := range file.Comments {
		var modified []*ast.Comment
		for i, c := range group.List {
			if pattern.MatchString(c.Text) {
				if modified == nil {
					modified = make([]*ast.Comment, i)
					copy(modified, group.List[0:i])
				}
			} else if modified != nil {
				modified = append(modified, c)
			}
		}
		if modified != nil {
			group.List = modified
		}
	}
}
