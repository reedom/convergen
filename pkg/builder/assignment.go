package builder

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"

	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/model"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
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

type assignmentBuilder struct {
	p         *FunctionBuilder
	methodPos token.Pos
	opts      option.Options
	src       srcStructEntry
}

func newAssignmentBuilder(p *FunctionBuilder, methodPos token.Pos, opts option.Options, src srcStructEntry) assignmentBuilder {
	return assignmentBuilder{
		p:         p,
		methodPos: methodPos,
		opts:      opts,
		src:       src,
	}
}

func (b assignmentBuilder) create(dst dstFieldEntry) (*model.Assignment, error) {
	lhs := dst.lhsExpr()

	if !dst.isFieldExported() {
		logger.Printf("%v: skip %v while it is not an exported field", b.p.fset.Position(b.methodPos), lhs)
		return nil, util.ErrNotFound
	}

	if b.opts.ShouldSkip(lhs) {
		logger.Printf("%v: skip %v", b.p.fset.Position(b.methodPos), lhs)
		return &model.Assignment{LHS: lhs, RHS: model.SkipField{}}, nil
	}

	var conv *option.FieldConverter
	for _, m := range b.opts.Converters {
		if m.Dst().Match(dst.fieldName(), true) {
			// If there are more than one converter exist for the dst, the last one wins.
			conv = m
		}
	}

	if conv != nil {
		return b.createWithConverter(dst, conv)
	} else {
		return b.createCommon(dst)
	}
}

func (b assignmentBuilder) buildRHS(srcObj types.Object, srcType, dstType types.Type) (string, bool) {
	if types.AssignableTo(srcType, dstType) {
		return b.src.rhsExpr(srcObj), true
	}

	if b.opts.Stringer && supportsStringer(srcType, dstType) {
		return b.src.rhsExpr(srcObj) + ".String()", true
	}

	if b.opts.Typecast && types.ConvertibleTo(srcType, dstType) {
		if rhs, ok := b.typeCast(dstType, b.src.rhsExpr(srcObj), b.methodPos); ok {
			return rhs, true
		}
	}
	return "", false
}

func (b assignmentBuilder) createCommon(dst dstFieldEntry) (*model.Assignment, error) {
	p := b.p
	opts := b.opts
	src := b.src
	methodPos := b.methodPos
	lhs := dst.lhsExpr()

	var mapper *option.NameMatcher
	for _, m := range opts.NameMapper {
		if m.Dst().Match(dst.fieldName(), true) {
			// If there are more than one mapper exist for the dst, the last one wins.
			mapper = m
		}
	}

	var a *model.Assignment
	var err error

	err = util.IterateMethods(src.strct.Type(), func(m *types.Func) (done bool, err error) {
		if src.IsPkgExternal() && !ast.IsExported(m.Name()) {
			return
		}

		retTypes, ok := util.GetMethodReturnTypes(m)
		if !ok || !compliesGetter(retTypes, false) {
			return
		}

		if mapper != nil {
			if !mapper.Src().Match(m.Name()+"()", true) {
				return
			}
		} else {
			if !opts.Getter || !opts.CompareFieldName(dst.fieldName(), m.Name()) {
				return
			}
		}

		retType := retTypes.At(0).Type()
		returnsError := retTypes.Len() == 2 && util.IsErrorType(retTypes.At(1).Type())
		if rhs, ok := b.buildRHS(m, retType, dst.fieldType()); ok {
			logger.Printf("%v: assignment found, %v to %v", p.fset.Position(methodPos), rhs, lhs)
			a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs, Error: returnsError}}
			return true, nil
		}

		return
	})
	if a != nil || err != nil {
		return a, err
	}

	if opts.Rule == model.MatchRuleName {
		err = util.IterateFields(src.strctType(), func(f *types.Var) (done bool, err error) {
			if src.IsPkgExternal() && !ast.IsExported(f.Name()) {
				return
			}

			if mapper != nil {
				if !mapper.Src().Match(f.Name(), true) {
					return
				}
			} else {
				if !opts.CompareFieldName(dst.fieldName(), f.Name()) {
					return
				}
			}

			if rhs, ok := b.buildRHS(f, f.Type(), dst.fieldType()); ok {
				logger.Printf("%v: assignment found, %v to %v", p.fset.Position(methodPos), rhs, lhs)
				a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs}}
				return true, nil
			}
			return
		})
	}
	if a != nil || err != nil {
		return a, err
	}

	logger.Printf("%v: no assignment for %v [%v]", p.fset.Position(methodPos), lhs, dst.fieldType().String())
	return &model.Assignment{LHS: lhs, RHS: model.NoMatchField{}}, nil
}

func (b assignmentBuilder) createWithConverter(dst dstFieldEntry, converter *option.FieldConverter) (*model.Assignment, error) {
	p := b.p
	opts := b.opts
	src := b.src
	methodPos := b.methodPos
	lhs := dst.lhsExpr()

	buildRHSWithConverter := func(srcObj types.Object, srcType types.Type) (string, bool) {
		arg, ok := b.buildRHS(srcObj, srcType, converter.ArgType())
		if !ok {
			return "", false
		}

		if types.AssignableTo(converter.RetType(), dst.fieldType()) {
			return converter.RHSExpr(arg), true
		}

		if opts.Stringer && supportsStringer(converter.RetType(), dst.fieldType()) {
			return converter.RHSExpr(arg + ".String()"), true
		}

		if opts.Typecast && types.ConvertibleTo(srcType, dst.fieldType()) {
			if expr, ok := b.typeCast(dst.fieldType(), arg, methodPos); ok {
				return converter.RHSExpr(expr), true
			}
		}
		return "", false
	}

	var a *model.Assignment
	var err error

	err = util.IterateMethods(src.strct.Type(), func(m *types.Func) (done bool, err error) {
		if src.IsPkgExternal() && !ast.IsExported(m.Name()) {
			return
		}

		retTypes, ok := util.GetMethodReturnTypes(m)
		if !ok || !compliesGetter(retTypes, false) {
			return
		}

		retType := retTypes.At(0).Type()
		if !converter.Src().Match(m.Name()+"()", true) {
			return
		}

		if rhs, ok := buildRHSWithConverter(m, retType); ok {
			logger.Printf("%v: assignment found, %v to %v", p.fset.Position(methodPos), rhs, lhs)
			a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs, Error: converter.ReturnsError()}}
			return true, nil
		}
		return
	})
	if a != nil || err != nil {
		return a, err
	}

	if opts.Rule == model.MatchRuleName {
		err = util.IterateFields(src.strctType(), func(f *types.Var) (done bool, err error) {
			if src.IsPkgExternal() && !ast.IsExported(f.Name()) {
				return
			}

			if !converter.Src().Match(f.Name(), true) {
				return
			}
			if rhs, ok := buildRHSWithConverter(f, f.Type()); ok {
				logger.Printf("%v: assignment found, %v to %v", p.fset.Position(methodPos), rhs, lhs)
				a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs, Error: converter.ReturnsError()}}
				return true, nil
			}
			return
		})
	}
	if a != nil || err != nil {
		return a, err
	}

	logger.Printf("%v: no assignment for %v [%v]", p.fset.Position(methodPos), lhs, dst.field.Type().String())
	return &model.Assignment{LHS: lhs, RHS: model.NoMatchField{}}, nil
}

func (b assignmentBuilder) typeCast(t types.Type, inner string, pos token.Pos) (string, bool) {
	switch typ := t.(type) {
	case *types.Pointer:
		return b.typeCast(typ.Elem(), inner, pos)
	case *types.Named:
		// If the type is defined within the current package.
		if b.p.pkg.Types.Scope().Lookup(typ.Obj().Name()) != nil {
			return fmt.Sprintf("%v(%v)", typ.Obj().Name(), inner), true
		}
		if pkgName, ok := b.p.imports.LookupName(typ.Obj().Pkg().Path()); ok {
			return fmt.Sprintf("%v.%v(%v)", pkgName, typ.Obj().Name(), inner), true
		}
		// TODO(reedom): add imports by code.
		logger.Printf("%v: cannot typecast as %v(%v) while the package %v is not imported",
			b.p.fset.Position(pos), typ.Obj().Name(), inner, typ.Obj().Pkg().Path())
		return "", false
	case *types.Basic:
		return fmt.Sprintf("%v(%v)", t.String(), inner), true
	default:
		logger.Printf("%v: typecast for %v is not implemented(yet) for %v",
			b.p.fset.Position(pos), t.String(), inner)
		return "", false
	}
}

func compliesGetter(retTypes *types.Tuple, returnsError bool) bool {
	num := retTypes.Len()
	if num == 0 || 2 < num {
		return false
	}
	return num == 1 || returnsError && util.IsErrorType(retTypes.At(1).Type())
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
