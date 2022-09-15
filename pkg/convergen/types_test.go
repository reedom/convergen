package convergen

import (
	"fmt"
	"go/types"
	"testing"
)

func TestWalkStruct(t *testing.T) {
	src := `
package main

type Model struct {
  ID   int
  Name string
  Cat *Category
}

type Category struct {
	ID   int
	Name string
}
`

	_, _, pkg := testLoadSrc(t, src)
	obj := pkg.Scope().Lookup("Model")
	opt := walkStructOpt{
		exactCase:     true,
		supportsError: false,
	}
	cb := func(pkg *types.Package, t types.Object, namePath string) (done bool, err error) {
		fmt.Printf("[cb] pkg=%v, namePath=%v, t=%#v\n", pkg.Name(), namePath, t)
		return
	}

	walkStruct("v", pkg, obj.Type(), cb, opt)
}
