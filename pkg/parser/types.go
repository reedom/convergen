package parser

import (
	"errors"
	"fmt"
	"go/types"
	"path"
	"strings"
)

var errNotFound = errors.New("not found")

func findInterface(scope *types.Scope, name string) *types.TypeName {
	typ := scope.Lookup(name)
	if typ == nil {
		return nil
	}
	intf, ok := typ.(*types.TypeName)
	if !ok {
		return nil
	}
	return intf
}

type lookupFieldOpt struct {
	exactCase     bool
	supportsError bool
	pattern       string
}

func findField(pkg *types.Package, t types.Type, opt lookupFieldOpt) (types.Object, error) {
	switch typ := t.(type) {
	case *types.Pointer:
		return findField(pkg, typ.Elem(), opt)
	case *types.Named:
		return findFieldInternal(pkg, typ.Obj().Pkg(), typ.Underlying(), opt, strings.Split(opt.pattern, "."))
	}
	return findFieldInternal(pkg, pkg, t, opt, strings.Split(opt.pattern, "."))
}

func findFieldInternal(pkg, typePkg *types.Package, t types.Type, opt lookupFieldOpt, pattern []string) (types.Object, error) {
	fmt.Printf("@@@ lookupFieldByPattern t: %v\n", t.String())
	if pattern[0] == "" {
		return nil, fmt.Errorf("invalid pattern")
	}

	switch typ := t.(type) {
	case *types.Pointer:
		return findFieldInternal(pkg, typePkg, typ.Elem(), opt, pattern)
	case *types.Named:
		for i := 0; i < typ.NumMethods(); i++ {
			m := typ.Method(i)
			if pkg.Name() != typePkg.Name() && !m.Exported() {
				continue
			}

			ok, err := pathMatch(pattern[0], m.Name(), opt.exactCase)
			if err != nil {
				return nil, err
			}
			if ok {
				if len(pattern) == 1 {
					return m, nil
				} else {
					return findFieldInternal(pkg, typePkg, m.Type(), opt, pattern[1:])
				}
			}
		}
		return findFieldInternal(pkg, typ.Obj().Pkg(), typ.Underlying(), opt, pattern)
	case *types.Struct:
		fmt.Printf("@@@ Struct: %v\n", typ.Underlying().String())
		for i := 0; i < typ.NumFields(); i++ {
			e := typ.Field(i)
			if pkg.Name() != typePkg.Name() && !e.Exported() {
				continue
			}

			ok, err := pathMatch(pattern[0], e.Name(), opt.exactCase)
			if err != nil {
				return nil, err
			}
			if ok {
				if len(pattern) == 1 {
					return e, nil
				} else {
					return findFieldInternal(pkg, typePkg, e.Type(), opt, pattern[1:])
				}
			}
		}
	}
	fmt.Printf("@@@ LAST: %#v, %v\n", pattern, t)
	return nil, errNotFound
}

func pathMatch(pattern, name string, exactCase bool) (bool, error) {
	if exactCase {
		return path.Match(pattern, name)
	}
	return path.Match(strings.ToLower(pattern), strings.ToLower(name))
}
