package convergen

import (
	"go/ast"
	"go/types"
	"regexp"
	"strings"

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
			if n.Doc != nil {
				return n.Doc
			}
		case *ast.FuncDecl:
			if n.Doc != nil {
				return n.Doc
			}
		case *ast.TypeSpec:
			if n.Doc != nil {
				return n.Doc
			}
		case *ast.Field:
			if n.Doc != nil {
				return n.Doc
			}
		}
	}
	return nil
}

// astRemoveMatchComments removes pattern matched comments from file.Comments.
func astRemoveMatchComments(file *ast.File, pattern *regexp.Regexp) {
	for _, group := range file.Comments {
		_ = astExtractMatchComments(group, pattern)
	}
}

// astExtractMatchComments removes pattern matched comments from commentGroup and return them.
func astExtractMatchComments(commentGroup *ast.CommentGroup, pattern *regexp.Regexp) []*ast.Comment {
	if commentGroup == nil {
		return nil
	}

	var modified, removed []*ast.Comment
	for i, c := range commentGroup.List {
		if pattern.MatchString(c.Text) {
			if modified == nil {
				removed = []*ast.Comment{commentGroup.List[i]}
				modified = make([]*ast.Comment, i)
				copy(modified, commentGroup.List[0:i])
			} else {
				removed = append(removed, commentGroup.List[i])
			}
		} else if modified != nil {
			modified = append(modified, c)
		}
	}
	if modified != nil {
		commentGroup.List = modified
	}
	return removed
}

type Imports map[string]string

func NewImports(specs []*ast.ImportSpec) Imports {
	imports := make(Imports)
	for _, spec := range specs {
		pkgPath := strings.ReplaceAll(spec.Path.Value, `"`, "")
		imports[pkgPath] = spec.Name.Name
	}
	return imports
}

func (i Imports) LookupName(pkgPath string) (name string, ok bool) {
	name, ok = i[pkgPath]
	return
}
