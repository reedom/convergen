package parser

import (
	"go/ast"
	"strings"
)

// importNames is a map of a package path to a package name in a convergen setup file.
type importNames map[string]string

// newImportNames creates a new importNames instance.
func newImportNames(specs []*ast.ImportSpec) importNames {
	imports := make(importNames)
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

// lookupName looks up the map with the pkgPath and returns its corresponding name
// in the conversion setup file.
func (i importNames) lookupName(pkgPath string) (name string, ok bool) {
	name, ok = i[pkgPath]
	return
}

// lookupPath looks up the map with the pkgName and returns its corresponding path
// in the conversion setup file.
func (i importNames) lookupPath(pkgName string) (path string, ok bool) {
	for p, n := range i {
		if n == pkgName {
			return p, true
		}
	}
	return
}
