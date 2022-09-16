package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
)

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

func astRemoveDecl(file *ast.File, name string) {
	comparer := func(s string) bool {
		return s == name
	}

	decls := make([]ast.Decl, 0)
	for _, decl := range file.Decls {
		if !ast.FilterDecl(decl, comparer) {
			decls = append(decls, decl)
		} else {
			fmt.Println("@@@ REMOVE")
		}
	}
	file.Decls = decls
}

func astRemoveDecl1(file *ast.File, fset *token.FileSet, node ast.Node) {
	decls := make([]ast.Decl, 0)
	nodePos := fset.Position(node.Pos()).String()
	for _, decl := range file.Decls {
		if fset.Position(decl.Pos()).String() != nodePos {
			decls = append(decls)
		}
	}
	file.Decls = decls
}

func astInsertComment(file *ast.File, text string, pos token.Pos) {
	comment := &ast.Comment{Slash: pos, Text: text}

	for i := range file.Comments {
		cg := file.Comments[i]
		if len(cg.List) == 0 {
			continue
		}

		if pos < cg.Pos() {
			e := &ast.CommentGroup{List: []*ast.Comment{comment}}
			file.Comments = append(file.Comments, e)
			copy(file.Comments[i+1:], file.Comments[i:])
			file.Comments[i] = e
			return
		}
		if cg.Pos() <= pos && pos < cg.End() {
			for j := range cg.List {
				if pos < cg.Pos() {
					cg.List = append(cg.List, comment)
					copy(cg.List[j+1:], cg.List[i:])
					cg.List[j] = comment
					return
				}
			}
			cg.List = append(cg.List, comment)
			return
		}
	}
	file.Comments = append(file.Comments, &ast.CommentGroup{List: []*ast.Comment{comment}})
}
