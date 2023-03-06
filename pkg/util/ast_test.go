package util_test

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"regexp"
	"strings"
	"testing"

	"github.com/reedom/convergen/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var reGoBuildGen = regexp.MustCompile(`\s*//\s*(go:(generate\b|build convergen\b)|\+build convergen)`)

func loadSrc(t *testing.T, src string) (*ast.File, *token.FileSet, *types.Package) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "example.go", src, parser.ParseComments)
	require.NoError(t, err, "failed to parse test src:\n%v", src)

	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("example.go", fset, []*ast.File{file}, nil)
	require.Nil(t, err)
	return file, fset, pkg
}

func getCodeText(t *testing.T, fset *token.FileSet, file *ast.File) string {
	buf := bytes.Buffer{}
	require.Nil(t, printer.Fprint(&buf, fset, file))
	return buf.String()
}

func TestRemoveMatchComments(t *testing.T) {
	t.Parallel()

	source := `
		package main

		// This is a comment.
		func main() {
			// This is another comment.
			fmt.Println("Hello, World!") // And this is a third comment.
		}
	`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	// Define the pattern to match comments that contain "another".
	pattern := regexp.MustCompile(`another`)

	// Call the function to remove matched comments.
	util.RemoveMatchComments(file, pattern)

	// Check that the matched comment was removed.
	for _, group := range file.Comments {
		for _, c := range group.List {
			if pattern.MatchString(c.Text) {
				t.Errorf("Expected comment '%s' to be removed, but it wasn't", c.Text)
			}
		}
	}
}

func TestMatchComments(t *testing.T) {
	t.Parallel()

	// Test with a nil comment group.
	result := util.MatchComments(nil, nil)
	if result {
		t.Error("Expected false, but got true")
	}

	// Test with an empty comment group.
	commentGroup := &ast.CommentGroup{}
	result = util.MatchComments(commentGroup, nil)
	if result {
		t.Error("Expected false, but got true")
	}

	// Test with a comment group that does not match.
	commentGroup = &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// This is a comment."},
			{Text: "// This is another comment."},
		},
	}
	pattern := regexp.MustCompile(`foo`)
	result = util.MatchComments(commentGroup, pattern)
	if result {
		t.Error("Expected false, but got true")
	}

	// Test with a comment group that matches.
	commentGroup = &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// This is a comment."},
			{Text: "// This is another comment."},
		},
	}
	pattern = regexp.MustCompile(`another`)
	result = util.MatchComments(commentGroup, pattern)
	if !result {
		t.Error("Expected true, but got false")
	}
}

func TestExtractMatchComments(t *testing.T) {
	// Define the comment group to test.
	commentGroup := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// This is a comment."},
			{Text: "// This is another comment."},
			{Text: "// This is a third comment."},
		},
	}

	// Define the pattern to match comments that contain "another".
	pattern := regexp.MustCompile(`another`)

	// Call the function to extract the matched comments.
	result := util.ExtractMatchComments(commentGroup, pattern)

	// Check that the matched comments were actually removed.
	if len(commentGroup.List) != 2 {
		t.Errorf("Expected 2 comments, but got %d", len(commentGroup.List))
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 comment, but got %d", len(result))
	}
	if result[0].Text != "// This is another comment." {
		t.Errorf("Expected comment '%s', but got '%s'", "// This is another comment.", result[0].Text)
	}
}

func TestRemoveDecl(t *testing.T) {
	// Define the source code to test.
	source := `
		package main

		import "fmt"

		func main() {
			fmt.Println("Hello, World!")
		}

		func foo() {
			fmt.Println("This is foo.")
		}

		func bar() {
			fmt.Println("This is bar.")
		}
	`
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "test.go", source, parser.AllErrors)
	assert.NoError(t, err)

	// Call the function to remove the "foo" declaration.
	util.RemoveDecl(file, "foo")

	// Check that the "foo" declaration was actually removed.
	foundFoo := false
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if funcDecl.Name.Name == "foo" {
			foundFoo = true
		}
	}
	assert.False(t, foundFoo, "Expected declaration 'foo' to be removed, but it wasn't")

	// Call the function to remove the "bar" declaration.
	util.RemoveDecl(file, "bar")

	// Check that the "bar" declaration was actually removed.
	foundBar := false
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if funcDecl.Name.Name == "bar" {
			foundBar = true
		}
	}
	assert.False(t, foundBar, "Expected declaration 'bar' to be removed, but it wasn't")

	// Check that the remaining declarations are correct.
	// Check that the "bar" declaration was actually removed.
	foundMain := false
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if funcDecl.Name.Name == "main" {
			foundMain = true
		}
	}
	assert.True(t, foundMain, "Expected declaration 'main' remains, but it doesn't")
}

func TestInsertComment(t *testing.T) {
	// Define the source code to test.
	source := `
		// This is the package comment.
		package main

		import "fmt"

		func main() {
			fmt.Println("Hello, World!")
		}
	`
	file, fset, _ := loadSrc(t, source)

	// Insert a comment at the beginning of the file.
	pos := file.Comments[0].Pos()
	util.InsertComment(file, "// This is a new comment.", pos)

	// Check that the comment was inserted correctly.
	expected := `
// This is the package comment.
// This is a new comment.
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
	`
	actual := getCodeText(t, fset, file)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))

	// Insert a comment at the end of the file.
	pos = file.End()
	util.InsertComment(file, "// This is the last comment.", pos)

	// Check that the comment was inserted correctly.
	expected = `
// This is the package comment.
// This is a new comment.
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}	// This is the last comment.
	`
	actual = getCodeText(t, fset, file)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
}

func TestRemoveBuildFlagAndGenerator(t *testing.T) {
	t.Parallel()

	src := `package main


//go:build convergen
// +build convergen

//go:generate go run github.com/reedom/convergen
var y = 1
`

	expected := `package main

var y = 1
`

	file, fset, _ := loadSrc(t, src)
	util.RemoveMatchComments(file, reGoBuildGen)
	actual := getCodeText(t, fset, file)
	assert.Equal(t, expected, actual)
}
