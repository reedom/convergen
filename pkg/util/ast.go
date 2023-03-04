package util

import (
	"go/ast"
	"go/token"
	"regexp"
)

// RemoveMatchComments removes pattern matched comments from file.Comments.
func RemoveMatchComments(file *ast.File, pattern *regexp.Regexp) {
	for _, group := range file.Comments {
		_ = ExtractMatchComments(group, pattern)
	}
}

// MatchComments reports whether any comment line in commentGroup contains
// any match of the regular expression pattern.
func MatchComments(commentGroup *ast.CommentGroup, pattern *regexp.Regexp) bool {
	if commentGroup == nil || len(commentGroup.List) == 0 {
		return false
	}

	for _, c := range commentGroup.List {
		if pattern.MatchString(c.Text) {
			return true
		}
	}
	return false
}

// ExtractMatchComments removes pattern matched comments from commentGroup and returns them.
func ExtractMatchComments(commentGroup *ast.CommentGroup, pattern *regexp.Regexp) []*ast.Comment {
	if commentGroup == nil || len(commentGroup.List) == 0 {
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

// RemoveDecl removes a declaration named name from file.Decls.
func RemoveDecl(file *ast.File, name string) {
	comparer := func(s string) bool {
		return s == name
	}

	decls := make([]ast.Decl, 0)
	for _, decl := range file.Decls {
		if !ast.FilterDecl(decl, comparer) {
			decls = append(decls, decl)
		}
	}
	file.Decls = decls
}

// InsertComment inserts a comment with the specified text at the specified
// position in file.Comments.
func InsertComment(file *ast.File, text string, pos token.Pos) {
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
