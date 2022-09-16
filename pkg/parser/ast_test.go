package parser

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLoadSrc(t *testing.T, src string) (*ast.File, *token.FileSet, *types.Package) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "example.go", src, parser.ParseComments)
	require.Nil(t, err, "failed to parse test src:\n%v", src)

	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("example.go", fset, []*ast.File{file}, nil)
	require.Nil(t, err)
	return file, fset, pkg
}

//func TestGetComments(t *testing.T) {
//	t.Parallel()
//
//	src := `
//package main
//
//// x is.
//var x = 0
//
//// Comment I-1
//// convergen:command i
//// Comment I-2
//type Convergen interface {
//   // Comment M-1
//	// convergen;command m
//	// Comment M-2
//	ToModel()
//}
//
//// y is.
//var y = 0
//`
//	file, _, pkg := testLoadSrc(t, src)
//	intf := findInterface(pkg.Scope(), "Convergen")
//	require.NotNil(t, intf)
//
//	found := getDocCommentOn(file, intf)
//	require.NotNil(t, found)
//	assert.Len(t, found.List, 3)
//	assert.Equal(t, "// Comment I-1", found.List[0].Text)
//}
//
//func TestGetEmptyComments(t *testing.T) {
//	t.Parallel()
//
//	src := `
//package main
//
//// x is.
//var x = 0
//
//type Convergen interface {
//   // Comment M-1
//	ToModel()
//}
//
//// y is.
//var y = 0
//`
//	file, _, pkg := testLoadSrc(t, src)
//	intf := findInterface(pkg.Scope(), "Convergen")
//	require.NotNil(t, intf)
//
//	found := getDocCommentOn(file, intf)
//	assert.Nil(t, found)
//}

func TestRemoveMatchComments(t *testing.T) {
	t.Parallel()

	src := `package main

// remain
// remove
// remain
var x = 1

// remove
// remain
var y = 1
`

	expected := `package main

// remain
// remain
var x = 1

// remain
var y = 1
`
	re := regexp.MustCompile(`//\s*remove\b`)

	file, fset, _ := testLoadSrc(t, src)
	astRemoveMatchComments(file, re)
	buf := bytes.Buffer{}
	require.Nil(t, printer.Fprint(&buf, fset, file))
	assert.Equal(t, expected, buf.String())
}
