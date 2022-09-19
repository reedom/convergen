package util

import (
	"go/ast"
	"strings"
)

// ImportNames is a map of a package path to a package name in a convergen setup file.
type ImportNames map[string]string

// NewImportNames creates a new ImportNames instance.
func NewImportNames(specs []*ast.ImportSpec) ImportNames {
	imports := make(ImportNames)
	for _, spec := range specs {
		pkgPath := strings.ReplaceAll(spec.Path.Value, `"`, "")
		if spec.Name != nil {
			imports[pkgPath] = spec.Name.Name
		} else {
			i := strings.LastIndex(pkgPath, "/")
			imports[pkgPath] = pkgPath[i+1:]
		}
	}
	return imports
}

// LookupName looks up the map with the pkgPath and returns its corresponding name
// in the conversion setup file.
func (i ImportNames) LookupName(pkgPath string) (name string, ok bool) {
	name, ok = i[pkgPath]
	return
}

// LookupPath looks up the map with the pkgName and returns its corresponding path
// in the conversion setup file.
func (i ImportNames) LookupPath(pkgName string) (path string, ok bool) {
	for p, n := range i {
		if n == pkgName {
			return p, true
		}
	}
	return
}
