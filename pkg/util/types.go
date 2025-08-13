// Package util provides utility functions for AST and type manipulation.
package util

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// LookupFieldOpt provides options for looking up a field.
type LookupFieldOpt struct {
	ExactCase     bool
	SupportsError bool
	Pattern       string
}

// ToAstNode converts types.Object to []ast.Node.
func ToAstNode(file *ast.File, obj types.Object) (path []ast.Node, exact bool) {
	if file == nil || obj == nil {
		return nil, false
	}

	// First try with the original file
	defer func() {
		if r := recover(); r != nil {
			// If PathEnclosingInterval panics, try with cleaned file
			if cleanedFile := safeCleanFileForAST(file); cleanedFile != nil {
				defer func() {
					if r2 := recover(); r2 != nil {
						// If cleaning still fails, return empty result
						path = nil
						exact = false
					}
				}()
				path, exact = astutil.PathEnclosingInterval(cleanedFile, obj.Pos(), obj.Pos())
			} else {
				path = nil
				exact = false
			}
		}
	}()

	return astutil.PathEnclosingInterval(file, obj.Pos(), obj.Pos())
}

// safeCleanFileForAST creates a safe copy of the file for AST processing.
// This prevents panics from empty comment groups without modifying the original.
func safeCleanFileForAST(file *ast.File) *ast.File {
	if file == nil {
		return nil
	}

	// Create a deep copy to avoid modifying the original
	cleanedFile := &ast.File{
		Doc:        file.Doc,
		Package:    file.Package,
		Name:       file.Name,
		Decls:      make([]ast.Decl, len(file.Decls)),
		Scope:      file.Scope,
		Imports:    file.Imports,
		Unresolved: file.Unresolved,
	}

	// Copy declarations
	copy(cleanedFile.Decls, file.Decls)

	// Filter out empty comment groups which cause panics in astutil
	if file.Comments != nil {
		validComments := make([]*ast.CommentGroup, 0, len(file.Comments))
		for _, cg := range file.Comments {
			if cg != nil && len(cg.List) > 0 {
				// Create a safe copy of the comment group
				safeCG := &ast.CommentGroup{
					List: make([]*ast.Comment, len(cg.List)),
				}
				copy(safeCG.List, cg.List)
				validComments = append(validComments, safeCG)
			}
		}
		cleanedFile.Comments = validComments
	} else {
		cleanedFile.Comments = nil
	}

	return cleanedFile
}

// IsErrorType returns true if the given type is an error type.
func IsErrorType(t types.Type) bool {
	return t.String() == "error"
}

// IsInvalidType returns true if the given type is an invalid type.
func IsInvalidType(t types.Type) bool {
	if typ, ok := DerefPtr(t).Underlying().(*types.Basic); ok {
		return typ.Kind() == types.Invalid
	}

	return false
}

// IsSliceType returns true if the given type is a slice type.
func IsSliceType(t types.Type) bool {
	_, ok := t.(*types.Slice)
	return ok
}

// IsBasicType returns true if the given type is a basic type.
func IsBasicType(t types.Type) bool {
	_, ok := t.(*types.Basic)
	return ok
}

// IsStructType returns true if the given type is a struct type.
func IsStructType(t types.Type) bool {
	_, ok := t.Underlying().(*types.Struct)
	return ok
}

// IsTypeParam returns true if the given type is a type parameter.
func IsTypeParam(t types.Type) bool {
	_, ok := t.(*types.TypeParam)
	return ok
}

// IsStructTypeOrTypeParam returns true if the given type is a struct type or a type parameter.
// This is useful for validation in generic contexts where type parameters will be resolved
// to concrete struct types at instantiation time.
func IsStructTypeOrTypeParam(t types.Type) bool {
	return IsStructType(t) || IsTypeParam(t)
}

// IsValidConversionType returns true if the given type is valid for conversion generation.
// This includes struct types, type parameters, and slices of type parameters (for variadic support).
func IsValidConversionType(t types.Type) bool {
	// Handle direct struct types and type parameters
	if IsStructType(t) || IsTypeParam(t) {
		return true
	}

	// Handle slices (for variadic parameters)
	if slice, ok := t.(*types.Slice); ok {
		// Allow slices of type parameters (e.g., []T for variadic ...T)
		return IsTypeParam(slice.Elem()) || IsStructType(slice.Elem())
	}

	return false
}

// IsNamedType returns true if the given type is a named type.
func IsNamedType(t types.Type) bool {
	_, ok := t.(*types.Named)
	return ok
}

// IsFunc returns true if the given type is a func type.
func IsFunc(obj types.Object) bool {
	_, ok := obj.(*types.Func)
	return ok
}

// IsPtr returns true if the given type is a pointer type.
func IsPtr(t types.Type) bool {
	_, ok := t.(*types.Pointer)
	return ok
}

// DerefPtr dereferences a *Pointer type and returns its base type.
func DerefPtr(typ types.Type) types.Type {
	if ptr, ok := typ.(*types.Pointer); ok {
		return ptr.Elem()
	}

	return typ
}

// Deref dereferences a type if it is a *Pointer type and returns its base type and true.
// Otherwise, it returns (typ, false).
func Deref(typ types.Type) (types.Type, bool) {
	if ptr, ok := typ.(*types.Pointer); ok {
		return ptr.Elem(), true
	}

	return typ, false
}

// PkgOf returns the package of the given type.
func PkgOf(t types.Type) *types.Package {
	switch typ := t.(type) {
	case *types.Pointer:
		return PkgOf(typ.Elem())
	case *types.Named:
		return typ.Obj().Pkg()
	default:
		return nil
	}
}

// SliceElement returns the type of the element in a slice type.
func SliceElement(t types.Type) types.Type {
	if slice, ok := t.(*types.Slice); ok {
		return slice.Elem()
	}

	return nil
}

// GetDocCommentOn retrieves doc comments that relate to nodes.
func GetDocCommentOn(file *ast.File, obj types.Object) (cg *ast.CommentGroup, cleanUp func()) {
	if file == nil || obj == nil {
		return nil, func() {}
	}

	// For interface methods, try specialized lookup first
	if methodObj, ok := obj.(*types.Func); ok {
		if signature, ok := methodObj.Type().(*types.Signature); ok {
			// For interface methods, the receiver is the interface type itself
			// We can detect interface methods by checking if the receiver is an interface
			if recv := signature.Recv(); recv != nil {
				if _, isInterface := recv.Type().Underlying().(*types.Interface); isInterface {
					// This is an interface method - try specialized lookup
					if comment := findInterfaceMethodComment(file, methodObj); comment != nil {
						return comment, createDocCleanupFunc(comment)
					}
				}
			}
		}
	}

	nodes, _ := ToAstNode(file, obj)
	if nodes == nil || len(nodes) == 0 {
		return nil, func() {}
	}

	for _, node := range nodes {
		if node == nil {
			continue
		}
		if docComment := extractDocComment(node); docComment != nil && len(docComment.List) > 0 {
			return docComment, createDocCleanupFunc(docComment)
		}
	}

	return nil, func() {}
}

// findInterfaceMethodComment searches for comments on interface method declarations.
// This is a specialized function for interface methods where ToAstNode fails to find the right nodes.
func findInterfaceMethodComment(file *ast.File, methodObj *types.Func) *ast.CommentGroup {
	methodName := methodObj.Name()
	methodPos := methodObj.Pos()

	// Walk through all declarations looking for interfaces
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
						// Look through interface methods
						for _, field := range interfaceType.Methods.List {
							// Check if this field represents our method
							for _, name := range field.Names {
								if name.Name == methodName {
									// Verify this is the right method by checking position proximity
									if isPositionClose(methodPos, name.Pos(), 10) {
										return field.Doc
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// isPositionClose checks if two token positions are reasonably close to each other.
// This helps match types.Object positions with AST node positions.
func isPositionClose(pos1, pos2 token.Pos, threshold int) bool {
	if pos1 == token.NoPos || pos2 == token.NoPos {
		return false
	}

	diff := int(pos1) - int(pos2)
	if diff < 0 {
		diff = -diff
	}

	return diff <= threshold
}

// extractDocComment extracts doc comment from various AST node types.
func extractDocComment(node ast.Node) *ast.CommentGroup {
	switch n := node.(type) {
	case *ast.GenDecl:
		return n.Doc
	case *ast.FuncDecl:
		return n.Doc
	case *ast.TypeSpec:
		return n.Doc
	case *ast.Field:
		return n.Doc
	case *ast.File:
		return n.Doc
	default:
		return nil
	}
}

// createDocCleanupFunc creates a cleanup function for the doc comment.
func createDocCleanupFunc(docGroup *ast.CommentGroup) func() {
	return func() {
		if docGroup == nil || len(docGroup.List) == 0 {
			// Note: We can't set the doc to nil here because we don't have
			// a reference to the original node field. This cleanup function
			// serves as a placeholder for more complex cleanup logic if needed.
		}
	}
}

// ToTextList returns a list of strings representing the comments in a CommentGroup.
func ToTextList(doc *ast.CommentGroup) []string {
	if doc == nil || len(doc.List) == 0 {
		return []string{}
	}

	list := make([]string, len(doc.List))
	for i := range doc.List {
		list[i] = doc.List[i].Text
	}

	return list
}

// PathMatch returns true if the name matches the pattern.
func PathMatch(pattern, name string, exactCase bool) (bool, error) {
	if exactCase {
		matched, err := path.Match(pattern, name)
		if err != nil {
			return false, fmt.Errorf("failed to match pattern %q against name %q: %w", pattern, name, err)
		}

		return matched, nil
	}

	matched, err := path.Match(strings.ToLower(pattern), strings.ToLower(name))
	if err != nil {
		return false, fmt.Errorf("failed to match pattern %q against name %q (case insensitive): %w", pattern, name, err)
	}

	return matched, nil
}

// FindMethod returns the method with the given name in the given type.
func FindMethod(t types.Type, name string, exactCase bool) (method *types.Func) {
	if !exactCase {
		name = strings.ToLower(name)
	}

	IterateMethods(t, func(m *types.Func) (done bool) {
		found := false
		if exactCase {
			found = m.Name() == name
		} else {
			found = strings.ToLower(m.Name()) == name
		}

		if found {
			method = m
		}

		return found
	})

	return
}

// IterateMethods iterates over the methods of the given type and calls the callback for each one.
func IterateMethods(t types.Type, cb func(*types.Func) (done bool)) {
	typ := DerefPtr(t)

	named, ok := typ.(*types.Named)
	if !ok {
		return
	}

	for i := 0; i < named.NumMethods(); i++ {
		m := named.Method(i)
		if cb(m) {
			return
		}
	}
}

// FindField returns the field with the given name from the given type.
func FindField(t types.Type, name string, exactCase bool) (field *types.Var) {
	if !exactCase {
		name = strings.ToLower(name)
	}

	IterateFields(t, func(f *types.Var) (done bool) {
		found := false
		if exactCase {
			found = f.Name() == name
		} else {
			found = strings.ToLower(f.Name()) == name
		}

		if found {
			field = f
		}

		return found
	})

	return
}

// IterateFields iterates over the fields of the given type and calls the callback for each one.
func IterateFields(t types.Type, cb func(*types.Var) (done bool)) {
	typ := DerefPtr(t)
	if named, ok := typ.(*types.Named); ok {
		typ = named.Underlying()
	}

	strct, ok := typ.Underlying().(*types.Struct)
	if !ok {
		return
	}

	for i := 0; i < strct.NumFields(); i++ {
		m := strct.Field(i)
		if cb(m) {
			return
		}
	}
}

// GetMethodReturnTypes returns the return types of the given method.
func GetMethodReturnTypes(m *types.Func) (*types.Tuple, bool) {
	sig := m.Type().(*types.Signature)

	num := sig.Results().Len()
	if num == 0 || 2 < num {
		return nil, false
	}

	return sig.Results(), true
}

// ParseGetterReturnTypes returns the return types of the given method.
func ParseGetterReturnTypes(m *types.Func) (ret types.Type, retError, ok bool) {
	sig := m.Type().(*types.Signature)

	num := sig.Results().Len()
	if num == 0 || 2 < num {
		return
	}

	if num == 2 {
		if !IsErrorType(sig.Results().At(1).Type()) {
			return
		}
	}

	return sig.Results().At(0).Type(), num == 2, true
}

// StringType returns the string type in the universe scope.
func StringType() types.Type {
	return types.Universe.Lookup("string").Type()
}

// CompliesGetter checks whether the given function complies with the requirements of a getter function.
// A getter function must have no input parameters and must return exactly one non-error value.
func CompliesGetter(m *types.Func) bool {
	sig := m.Type().(*types.Signature)
	if sig.Params().Len() != 0 {
		return false
	}

	num := sig.Results().Len()

	return num == 1 && !IsErrorType(sig.Results().At(0).Type())
}

// CompliesStringer checks if the given type is a Stringer compliant type,
// which has a method "String()" that takes no arguments and returns a string.
func CompliesStringer(src types.Type) bool {
	named, ok := DerefPtr(src).(*types.Named)
	if !ok {
		return false
	}

	obj, _, _ := types.LookupFieldOrMethod(named, false, named.Obj().Pkg(), "String")
	if obj == nil {
		return false
	}

	sig, ok := obj.Type().(*types.Signature)
	if !ok {
		return false
	}

	return sig.Params().Len() == 0 &&
		sig.Results().Len() == 1 &&
		sig.Results().At(0).Type().String() == "string"
}
