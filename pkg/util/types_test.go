package util_test

import (
	"testing"

	"github.com/reedom/convergen/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestIsErrorType(t *testing.T) {
	src := `package main
var err error
type ErrFoo error
var err2 ErrFoo`

	_, _, pkg := loadSrc(t, src)

	obj := pkg.Scope().Lookup("err")
	assert.True(t, util.IsErrorType(obj.Type()))

	obj = pkg.Scope().Lookup("err2")
	assert.False(t, util.IsErrorType(obj.Type()))
}
