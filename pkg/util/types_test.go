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

func TestDerefPtr(t *testing.T) {
	// Test DerefPtr with basic types.
	typ := types.Typ[types.Int]
	assert.Equal(t, typ, util.DerefPtr(typ))

	// Test DerefPtr with named types.
	obj := types.NewTypeName(0, nil, "MyInt", types.Typ[types.Int])
	named := types.NewNamed(obj, types.Typ[types.Int], nil)
	assert.Equal(t, named, util.DerefPtr(named))

	// Test DerefPtr with pointer types.
	ptrTyp := types.NewPointer(typ)
	assert.Equal(t, typ, util.DerefPtr(ptrTyp))

	// Test DerefPtr with nested pointer types.
	nestedPtrTyp := types.NewPointer(ptrTyp)
	assert.Equal(t, ptrTyp, util.DerefPtr(nestedPtrTyp))
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
