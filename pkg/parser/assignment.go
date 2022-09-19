package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"

	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/model"
	"github.com/reedom/convergen/pkg/parser/option"
	"golang.org/x/tools/go/loader"
)

type dstFieldEntry struct {
	model.Var
	field *types.Var
}

func (f dstFieldEntry) fieldName() string {
	return f.field.Name()
}

func (f dstFieldEntry) fieldType() types.Type {
	return f.field.Type()
}

func (f dstFieldEntry) lhsExpr() string {
	return fmt.Sprintf("%v.%v", f.Name, f.fieldName())
}

func (f dstFieldEntry) isFieldExported() bool {
	return f.PkgName == "" || ast.IsExported(f.fieldName())
}

type srcStructEntry struct {
	model.Var
	strct *types.Var
}

func (s srcStructEntry) strctType() types.Type {
	return s.strct.Type()
}

func (s srcStructEntry) rhsExpr(obj types.Object) string {
	switch obj.(type) {
	case *types.Var:
		return fmt.Sprintf("%v.%v", s.Name, obj.Name())
	case *types.Func:
		return fmt.Sprintf("%v.%v()", s.Name, obj.Name())
	default:
		panic(fmt.Sprintf("not implemented for %#v", obj))
	}
}

func (p *Parser) createAssign(methodPos token.Pos, opts options, dst dstFieldEntry, src srcStructEntry) (*model.Assignment, error) {
	lhs := dst.lhsExpr()

	if !dst.isFieldExported() {
		logger.Printf("%v: skip %v while it is not an exported field", p.fset.Position(methodPos), lhs)
		return nil, errNotFound
	}

	if opts.shouldSkip(lhs) {
		logger.Printf("%v: skip %v", p.fset.Position(methodPos), lhs)
		return &model.Assignment{LHS: lhs, RHS: model.SkipField{}}, nil
	}

	var mapper *option.NameMatcher
	for _, m := range opts.nameMapper {
		if m.Dst().Match(dst.fieldName(), opts.exactCase) {
			// If there are more than one mapper exist for the dst, the last one wins.
			mapper = m
		}
	}

	var a *model.Assignment
	var err error
	inner := func(f types.Object, t types.Type) {
		if types.AssignableTo(t, dst.fieldType()) {
			lhs := dst.lhsExpr()
			rhs := src.rhsExpr(f)
			logger.Printf("%v: assignment found, %v to %v [%v]", p.fset.Position(methodPos), rhs, lhs, dst.fieldType().String())
			a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs}}
			return
		}

		if opts.stringer && supportsStringer(t, dst.fieldType()) {
			lhs := dst.lhsExpr()
			rhs := src.rhsExpr(f) + ".String()"
			logger.Printf("%v: assignment found, %v to %v", p.fset.Position(methodPos), rhs, lhs)
			a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs}}
			return
		}

		if opts.typecast && types.ConvertibleTo(t, dst.fieldType()) {
			lhs := dst.lhsExpr()
			if rhs, ok := p.typeCast(dst.fieldType(), src.rhsExpr(f), methodPos); ok {
				logger.Printf("%v: assignment found, %v to %v", p.fset.Position(methodPos), rhs, lhs)
				a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs}}
				return
			}
		}
	}

	if opts.getter {
		err = iterateMethods(src.strct.Type(), func(m *types.Func) (done bool, err error) {
			retTypes, ok := getMethodReturnTypes(m)
			if !ok || !compliesGetter(retTypes, false) {
				return
			}

			if mapper != nil {
				if !mapper.Src().Match(m.Name()+"()", opts.exactCase) {
					return
				}
			} else if !opts.compareFieldName(dst.fieldName(), m.Name()) {
				return
			}
			if src.IsPkgExternal() && !ast.IsExported(m.Name()) {
				return
			}

			retType := retTypes.At(0).Type()
			inner(m, retType)
			return a != nil, nil
		})
	}
	if a != nil || err != nil {
		return a, err
	}

	if opts.rule == model.MatchRuleName {
		err = iterateFields(src.strctType(), func(f *types.Var) (done bool, err error) {
			if mapper != nil {
				if !mapper.Src().Match(f.Name(), opts.exactCase) {
					return
				}
			} else if !opts.compareFieldName(dst.fieldName(), f.Name()) {
				return
			}

			if src.IsPkgExternal() && !ast.IsExported(f.Name()) {
				return
			}

			inner(f, f.Type())
			return a != nil, nil
		})
	}
	if a != nil || err != nil {
		return a, err
	}

	logger.Printf("%v: no assignment for %v [%v]", p.fset.Position(methodPos), lhs, dst.field.Type().String())
	return &model.Assignment{LHS: lhs, RHS: model.NoMatchField{}}, nil
}

func (p *Parser) createAssignFromFields(methodPos token.Pos, opts options, dst dstFieldEntry, src srcStructEntry) (*model.Assignment, error) {
	var a *model.Assignment

	err := iterateFields(src.strctType(), func(f *types.Var) (done bool, err error) {
		if !opts.compareFieldName(dst.fieldName(), f.Name()) {
			return
		}
		if src.IsPkgExternal() && !ast.IsExported(f.Name()) {
			return
		}

		if types.AssignableTo(f.Type(), dst.fieldType()) {
			lhs := dst.lhsExpr()
			rhs := src.rhsExpr(f)
			logger.Printf("%v: assignment found, %v to %v [%v]", p.fset.Position(methodPos), rhs, lhs, dst.fieldType().String())
			a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs}}
			return true, nil
		}

		if opts.stringer && supportsStringer(f.Type(), dst.fieldType()) {
			lhs := dst.lhsExpr()
			rhs := src.rhsExpr(f) + ".String()"
			logger.Printf("%v: assignment found, %v to %v", p.fset.Position(methodPos), rhs, lhs)
			a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs}}
			return true, nil
		}

		if opts.typecast && types.ConvertibleTo(f.Type(), dst.fieldType()) {
			lhs := dst.lhsExpr()
			if rhs, ok := p.typeCast(f.Type(), src.rhsExpr(f), methodPos); ok {
				logger.Printf("%v: assignment found, %v to %v", p.fset.Position(methodPos), rhs, lhs)
				a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs}}
			}
			return true, nil
		}
		return
	})

	return a, err
}

func (p *Parser) typeCast(t types.Type, inner string, pos token.Pos) (string, bool) {
	switch typ := t.(type) {
	case *types.Pointer:
		return p.typeCast(typ.Elem(), inner, pos)
	case *types.Named:
		// If the type is defined within the current package.
		if p.pkg.Types.Scope().Lookup(typ.Obj().Name()) != nil {
			return fmt.Sprintf("%v(%v)", typ.Obj().Name(), inner), true
		}
		if pkgName, ok := p.imports.lookupName(typ.Obj().Pkg().Path()); ok {
			return fmt.Sprintf("%v.%v(%v)", pkgName, typ.Obj().Name(), inner), true
		}
		// TODO(reedom): add imports by code.
		logger.Printf("%v: cannot typecast as %v(%v) while the package %v is not imported",
			p.fset.Position(pos), typ.Obj().Name(), inner, typ.Obj().Pkg().Path())
		return "", false
	case *types.Basic:
		return fmt.Sprintf("%v(%v)", t.String(), inner), true
	default:
		logger.Printf("%v: typecast for %v is not implemented(yet) for %v",
			p.fset.Position(pos), t.String(), inner)
		return "", false
	}
}

func compliesGetter(retTypes *types.Tuple, returnsError bool) bool {
	num := retTypes.Len()
	if num == 0 || 2 < num {
		return false
	}
	return num == 1 || returnsError && isErrorType(retTypes.At(1).Type())
}

var stringer *types.Interface

func supportsStringer(src types.Type, dst types.Type) bool {
	strType := types.Universe.Lookup("string").Type()
	if !types.AssignableTo(strType, dst) {
		return false
	}

	if stringer == nil {
		conf := loader.Config{ParserMode: parser.ParseComments}
		conf.Import("fmt")
		lprog, _ := conf.Load()
		t := lprog.Package("fmt").Pkg.Scope().Lookup("Stringer").Type()
		stringer = t.Underlying().(*types.Interface)
	}

	return types.Implements(src, stringer)
}
