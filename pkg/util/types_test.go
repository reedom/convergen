package util_test

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/reedom/convergen/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToAstNode(t *testing.T) {
	t.Parallel()
	// Define the source code to test.
	source := `
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
`
	file, _, pkg := loadSrc(t, source)

	// Test ToAstNode with a package-level variable.
	obj := pkg.Scope().Lookup("main")
	path, exact := util.ToAstNode(file, obj)
	assert.True(t, exact)
	assert.Equal(t, 3, len(path))
	assert.IsType(t, &ast.Ident{}, path[0])
	assert.IsType(t, &ast.FuncDecl{}, path[1])
	funcDecl := path[1].(*ast.FuncDecl)
	assert.Equal(t, "main", funcDecl.Name.Name)
}

func TestIsErrorType(t *testing.T) {
	t.Parallel()
	src := `
package main

var err error
type ErrFoo error
var err2 ErrFoo
`
	_, _, pkg := loadSrc(t, src)

	obj := pkg.Scope().Lookup("err")
	assert.True(t, util.IsErrorType(obj.Type()))

	obj = pkg.Scope().Lookup("err2")
	assert.False(t, util.IsErrorType(obj.Type()))
}

func TestIsInvalidType(t *testing.T) {
	t.Parallel()
	src := `
package custom

var MyInt int
`
	_, _, pkg := loadSrc(t, src)

	obj := pkg.Scope().Lookup("MyInt")
	assert.False(t, util.IsInvalidType(obj.Type()))
}

func TestIsSliceType(t *testing.T) {
	t.Parallel()
	src := `
package custom

var MySlice []int
var MyVar int
`
	_, _, pkg := loadSrc(t, src)

	obj := pkg.Scope().Lookup("MySlice")
	assert.True(t, util.IsSliceType(obj.Type()))
	obj = pkg.Scope().Lookup("MyVar")
	assert.False(t, util.IsSliceType(obj.Type()))
}

func TestIsBasicType(t *testing.T) {
	t.Parallel()
	src := `
package custom

type MyInt int
type MyMyInt MyInt
var myInt MyMyInt
`
	_, _, pkg := loadSrc(t, src)

	obj := pkg.Scope().Lookup("myInt")
	assert.False(t, util.IsBasicType(obj.Type()))
	assert.True(t, util.IsBasicType(obj.Type().Underlying()))
}

func TestIsStructType(t *testing.T) {
	t.Parallel()
	src := `
package custom

type MyStruct struct {}
var myStruct MyStruct
var RawStruct struct {}
`
	_, _, pkg := loadSrc(t, src)

	obj := pkg.Scope().Lookup("myStruct")
	assert.True(t, util.IsStructType(obj.Type()))
	obj = pkg.Scope().Lookup("RawStruct")
	assert.True(t, util.IsStructType(obj.Type()))
}

func TestIsNamedType(t *testing.T) {
	t.Parallel()
	src := `
package custom

type MyStruct struct {}
var myStruct MyStruct
var RawStruct struct {}
`
	_, _, pkg := loadSrc(t, src)

	obj := pkg.Scope().Lookup("myStruct")
	assert.True(t, util.IsNamedType(obj.Type()))
	obj = pkg.Scope().Lookup("RawStruct")
	assert.False(t, util.IsNamedType(obj.Type()))
}

func TestIsFunc(t *testing.T) {
	t.Parallel()
	src := `
package custom

var myFunc func()

func main() {}
`
	_, _, pkg := loadSrc(t, src)

	obj := pkg.Scope().Lookup("myFunc")
	assert.False(t, util.IsFunc(obj))
	obj = pkg.Scope().Lookup("main")
	assert.True(t, util.IsFunc(obj))
}

func TestIsPtr(t *testing.T) {
	// Test IsPtr with basic types.
	typ := types.Typ[types.Int]
	assert.False(t, util.IsPtr(typ))

	// Test IsPtr with named types.
	obj := types.NewTypeName(0, nil, "MyInt", types.Typ[types.Int])
	named := types.NewNamed(obj, types.Typ[types.Int], nil)
	assert.False(t, util.IsPtr(named))

	// Test IsPtr with pointer types.
	ptrTyp := types.NewPointer(typ)
	assert.True(t, util.IsPtr(ptrTyp))

	// Test IsPtr with nested pointer types.
	nestedPtrTyp := types.NewPointer(ptrTyp)
	assert.True(t, util.IsPtr(nestedPtrTyp))
}

func TestDeref(t *testing.T) {
	t.Parallel()

	typ := types.Typ[types.Int]
	// Test DerefPtr with basic types.
	assert.Equal(t, typ, util.DerefPtr(typ))
	// Test Deref with basic types.
	actual, ok := util.Deref(typ)
	assert.False(t, ok)
	assert.Equal(t, typ, actual)

	obj := types.NewTypeName(0, nil, "MyInt", types.Typ[types.Int])
	named := types.NewNamed(obj, types.Typ[types.Int], nil)
	// Test DerefPtr with named types.
	assert.Equal(t, named, util.DerefPtr(named))
	// Test Deref with named types.
	actual, ok = util.Deref(named)
	assert.False(t, ok)
	assert.Equal(t, named, actual)

	ptrTyp := types.NewPointer(typ)
	// Test DerefPtr with pointer types.
	assert.Equal(t, typ, util.DerefPtr(ptrTyp))
	// Test Deref with pointer types.
	actual, ok = util.Deref(ptrTyp)
	assert.True(t, ok)
	assert.Equal(t, typ, actual)

	nestedPtrTyp := types.NewPointer(ptrTyp)
	// Test DerefPtr with nested pointer types.
	assert.Equal(t, ptrTyp, util.DerefPtr(nestedPtrTyp))
	// Test Deref with pointer types.
	actual, ok = util.Deref(nestedPtrTyp)
	assert.True(t, ok)
	assert.Equal(t, ptrTyp, actual)
}

func TestPkgOf(t *testing.T) {
	t.Parallel()

	// Define types in the "testing" and "fmt" packages.
	testingPkg := types.NewPackage("testing", "testing")
	fmtPkg := types.NewPackage("fmt", "fmt")
	testingIntTyp := types.NewNamed(types.NewTypeName(0, testingPkg, "Int", types.Typ[types.Int]), types.Typ[types.Int], nil)
	fmtStringTyp := types.NewNamed(types.NewTypeName(0, fmtPkg, "String", types.Typ[types.String]), types.Typ[types.String], nil)

	// Define a basic type and a named type.
	typ := types.Typ[types.Int]
	namedTyp := types.NewNamed(types.NewTypeName(0, nil, "MyInt", types.Typ[types.Int]), types.Typ[types.Int], nil)

	// Define pointer types for the basic type, named type, and testing package type.
	ptrTyp := types.NewPointer(typ)
	namedPtrTyp := types.NewPointer(namedTyp)
	testingPtrTyp := types.NewPointer(testingIntTyp)

	// Test PkgOf with basic types and named types.
	assert.Nil(t, util.PkgOf(typ))
	assert.Nil(t, util.PkgOf(namedTyp))

	// Test PkgOf with pointer types.
	assert.Nil(t, util.PkgOf(ptrTyp))
	assert.Nil(t, util.PkgOf(namedPtrTyp))
	assert.Equal(t, testingPkg, util.PkgOf(testingPtrTyp))

	// Test PkgOf with a named type defined in the "fmt" package.
	assert.Equal(t, fmtPkg, util.PkgOf(fmtStringTyp))
}

func TestSliceElement(t *testing.T) {
	// Define a basic type and a named type.
	typ := types.Typ[types.Int]
	namedTyp := types.NewNamed(types.NewTypeName(0, nil, "MyInt", types.Typ[types.Int]), types.Typ[types.Int], nil)

	// Define a slice type for the basic type and the named type.
	sliceTyp := types.NewSlice(typ)
	namedSliceTyp := types.NewSlice(namedTyp)

	// Test SliceElement with basic types and named types.
	assert.Nil(t, util.SliceElement(typ))
	assert.Nil(t, util.SliceElement(namedTyp))

	// Test SliceElement with slice types.
	assert.Equal(t, typ, util.SliceElement(sliceTyp))
	assert.Equal(t, namedTyp, util.SliceElement(namedSliceTyp))
}

func TestGetDocCommentOn(t *testing.T) {
	t.Parallel()
	// Define the source code to test.
	source := `
// package comment
package main

// MyReader comment
type MyReader interface {
	// method comment
	Read()
}

// MyInt comment
type MyInt int

// func comment
func main() {}
`
	file, _, pkg := loadSrc(t, source)

	// *ast.GenDecl
	obj := pkg.Scope().Lookup("MyReader")
	cg, cleanUp := util.GetDocCommentOn(file, obj)
	if assert.NotEmpty(t, cg) {
		assert.Equal(t, "MyReader comment\n", cg.Text())
	}
	cleanUp()

	// *ast.Field
	iface := obj.Type().Underlying().(*types.Interface)
	obj = iface.Method(0)
	cg, cleanUp = util.GetDocCommentOn(file, obj)
	if assert.NotEmpty(t, cg) {
		assert.Equal(t, "method comment\n", cg.Text())
	}
	cleanUp()

	// *ast.FuncDecl
	obj = pkg.Scope().Lookup("main")
	cg, cleanUp = util.GetDocCommentOn(file, obj)
	if assert.NotEmpty(t, cg) {
		assert.Equal(t, "func comment\n", cg.Text())
	}
	cleanUp()

	// *ast.FuncDecl
	obj = pkg.Scope().Lookup("MyInt")
	cg, cleanUp = util.GetDocCommentOn(file, obj)
	if assert.NotEmpty(t, cg) {
		assert.Equal(t, "MyInt comment\n", cg.Text())
	}
	cleanUp()
}

func TestToTextList(t *testing.T) {
	t.Parallel()
	assert.Empty(t, util.ToTextList(nil))

	doc := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// comment 1\n"},
			{Text: "// comment 2\n"},
		},
	}
	actual := util.ToTextList(doc)
	assert.Equal(t, []string{"// comment 1\n", "// comment 2\n"}, actual)
}

//func TestWalkStruct(t *testing.T) {
//	src := `
//package main
//
//type Model struct {
//  ID   int
//  Name string
//  Cat *Category
//}
//
//type Category struct {
//	ID   int
//	Name string
//}
//`
//
//	_, _, pkg := testLoadSrc(t, src)
//	obj := pkg.Scope().Lookup("Model")
//	opt := walkStructOpt{
//		exactCase:     true,
//		supportsError: false,
//	}
//	cb := func(pkg *types.Package, t types.Object, namePath string) (done bool, err error) {
//		fmt.Printf("[cb] pkg=%v, namePath=%v, t=%#v\n", pkg.Name(), namePath, t)
//		return
//	}
//
//	walkStruct("v", pkg, obj.structObj(), cb, opt)
//}

func TestPathMatch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pattern   string
		path      string
		exactCase bool
		match     bool
	}{
		{"Name", "Name", true, true},
		{"Name", "Nam", true, false},
		{"Name", "name", true, false},
		{"name", "Name", true, false},
		{"name", "Name", false, true},
		{"Name", "name", false, true},
		{"Na*", "name", false, true},
		{"N*e", "name", false, true},
	}

	for i, tt := range cases {
		actual, err := util.PathMatch(tt.pattern, tt.path, tt.exactCase)
		require.Nil(t, err, `case %v has invalid pattern "%v"`, i, tt.pattern)
		if tt.match {
			assert.True(t, actual, `pattern "%v" against "%v" should match`)
		} else {
			assert.False(t, actual, `pattern "%v" against "%v" should not match`)
		}
	}
}

func TestFindMethod(t *testing.T) {
	t.Parallel()
	// Define the source code to test.
	source := `
package main

type I interface {
	Foo()
	Bar()
	Baz() error
}

type S struct{}

func (s S) Foo() {}
func (s S) Bar() {}
func (s S) Baz() error {
	return nil
}
`
	_, _, pkg := loadSrc(t, source)

	obj := pkg.Scope().Lookup("S")

	// Find the method named "Bar" (exact case).
	m := util.FindMethod(obj.Type(), "Bar", true)
	assert.Equal(t, "Bar", m.Name())

	// Find the method named "foo" (case insensitive).
	m = util.FindMethod(obj.Type(), "foo", false)
	assert.Equal(t, "Foo", m.Name())
}

func TestFindField(t *testing.T) {
	t.Parallel()
	// Define the source code to test.
	source := `
package main

type S struct{
 	Foo int
 	Bar string
}
`
	_, _, pkg := loadSrc(t, source)

	// *ast.GenDecl
	obj := pkg.Scope().Lookup("S")

	// Find the method named "Bar" (exact case).
	f := util.FindField(obj.Type(), "Bar", true)
	assert.Equal(t, "Bar", f.Name())

	// Find the method named "foo" (case insensitive).
	f = util.FindField(obj.Type(), "foo", false)
	assert.Equal(t, "Foo", f.Name())
}

func TestGetMethodReturnTypes(t *testing.T) {
	t.Parallel()
	// Define the source code to test.
	source := `
package main

func Foo() {}
func Bar() error {
	return nil
}
func Baz() (int, error) {
	return 0, nil
}
`
	_, _, pkg := loadSrc(t, source)

	fn := pkg.Scope().Lookup("Foo")
	_, ok := util.GetMethodReturnTypes(fn.(*types.Func))
	assert.False(t, ok)

	fn = pkg.Scope().Lookup("Bar")
	ret, ok := util.GetMethodReturnTypes(fn.(*types.Func))
	if assert.True(t, ok) {
		assert.Equal(t, 1, ret.Len())
		assert.True(t, util.IsErrorType(ret.At(0).Type()))
	}

	fn = pkg.Scope().Lookup("Baz")
	ret, ok = util.GetMethodReturnTypes(fn.(*types.Func))
	if assert.True(t, ok) {
		assert.Equal(t, 2, ret.Len())
	}
}

func TestCompliesGetter(t *testing.T) {
	t.Parallel()
	// Define the source code to test.
	source := `
package main

type I interface {
	Proc()
	Getter() int
	GetterWithError() (int, error)
	NonGetter(i int) int
}

type S struct{}

func (s S) Proc() {}
func (s S) Getter() int {
	return 0
}
func (s S) FuncWithError() (int, error) {
	return 0, nil
}
func (s S) NonGetter(i int) int {
	return 0
}
`
	_, _, pkg := loadSrc(t, source)

	obj := pkg.Scope().Lookup("S")

	cases := []struct {
		name     string
		expected bool
	}{
		{"Proc", false},
		{"Getter", true},
		{"FuncWithError", false},
		{"NonGetter", false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			m := util.FindMethod(obj.Type(), tt.name, true)
			assert.Equal(t, tt.expected, util.CompliesGetter(m))
		})
	}
}

func TestCompliesStringer(t *testing.T) {
	t.Parallel()
	// Define the source code to test.
	source := `
package main

type S struct{}

func (s S) String() string {
	return ""
}

type T struct{}

func (t T) GetString() string {
	return ""
}

type U struct {
	String string
}

func V() string {
	return ""
}
`
	_, _, pkg := loadSrc(t, source)

	cases := []struct {
		name     string
		expected bool
	}{
		{"S", true},
		{"T", false},
		{"U", false},
		{"V", false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			obj := pkg.Scope().Lookup(tt.name)
			assert.Equal(t, tt.expected, util.CompliesStringer(obj.Type()))
		})
	}
}
