package convergen

import (
	"errors"
	"fmt"
	"go/types"
	"path"
	"strings"
)

var errNotFound = errors.New("not found")

type lookupFieldOpt struct {
	exactCase     bool
	supportsError bool
	pattern       string
}

func isErrorType(t types.Type) bool {
	return t.String() == "error"
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
	case *types.Basic:
	case *types.Array:
	case *types.Slice:
	case *types.Map:
	case *types.Chan:
	case *types.Interface:
	//case *types.Tuple:
	case *types.Signature:
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

type walkStructCallback = func(pkg *types.Package, obj types.Object, namePath string) (done bool, err error)

type walkStructOpt struct {
	exactCase     bool
	supportsError bool
}

type walkStructWork struct {
	varName   string
	pkg       *types.Package
	cb        walkStructCallback
	isPointer bool
	namePath  namePathType
	done      *bool
}

type namePathType string

func (n namePathType) String() string {
	return string(n)
}

func (n namePathType) add(name string) namePathType {
	return namePathType(n.String() + "." + name)
}

func (w walkStructWork) withNamePath(namePath namePathType) walkStructWork {
	work := w
	work.namePath = namePath
	return work
}

func walkStruct(varName string, pkg *types.Package, t types.Type, cb walkStructCallback, opt walkStructOpt) error {
	done := false
	work := walkStructWork{
		varName:   varName,
		pkg:       pkg,
		cb:        cb,
		isPointer: false,
		namePath:  namePathType(varName),
		done:      &done,
	}

	switch typ := t.(type) {
	case *types.Pointer:
		return walkStruct(varName, pkg, typ.Elem(), cb, opt)
	case *types.Named:
		return walkStructInternal(typ.Obj().Pkg(), typ.Underlying(), opt, work)
	}

	return walkStructInternal(pkg, t, opt, work)
}

func walkStructInternal(typePkg *types.Package, t types.Type, opt walkStructOpt, work walkStructWork) error {
	//isPointer := work.isPointer
	work.isPointer = false

	switch typ := t.(type) {
	case *types.Pointer:
		work.isPointer = true
		return walkStructInternal(typePkg, typ.Elem(), opt, work)
	case *types.Named:
		return walkStructInternal(typ.Obj().Pkg(), typ.Underlying(), opt, work)
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			f := typ.Field(i)
			namePath := work.namePath.add(f.Name())
			done, err := work.cb(typePkg, f, namePath.String())
			if err != nil {
				return err
			}
			if done {
				continue
			}

			// Down to the f.Type() hierarchy.
			err = walkStructInternal(typePkg, f.Type(), opt, work.withNamePath(namePath))
			if err != nil {
				return err
			}
		}
		return nil
	}

	fmt.Printf("@@@ %#v\n", t)
	return nil
}
