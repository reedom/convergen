package builder

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"

	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/model"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
	"golang.org/x/tools/go/loader"
)

type dstFieldEntry struct {
	parent *dstFieldEntry
	model.Var
	field *types.Var
}

func (f dstFieldEntry) fieldName() string {
	return f.field.Name()
}

func (f dstFieldEntry) fieldPath() string {
	if f.parent != nil {
		return fmt.Sprintf("%v.%v", f.parent.fieldPath(), f.field.Name())
	}
	return f.field.Name()
}

func (f dstFieldEntry) fieldType() types.Type {
	return f.field.Type()
}

func (f dstFieldEntry) lhsExpr() string {
	if f.parent != nil {
		return fmt.Sprintf("%v.%v", f.parent.lhsExpr(), f.fieldName())
	}
	return fmt.Sprintf("%v.%v", f.Name, f.fieldName())
}

func (f dstFieldEntry) isFieldExported() bool {
	return f.PkgName == "" || ast.IsExported(f.fieldName())
}

type srcStructEntry struct {
	parent *srcStructEntry
	model.Var
	strct *types.Var
}

func (s srcStructEntry) strctType() types.Type {
	return s.strct.Type()
}

func (s srcStructEntry) root() srcStructEntry {
	if s.parent != nil {
		return s.parent.root()
	}
	return s
}

func (s srcStructEntry) fieldPath(obj types.Object) string {
	switch obj.(type) {
	case *types.Var:
		if s.parent != nil {
			return fmt.Sprintf("%v.%v", s.parent.fieldPath(s.strct), obj.Name())
		}
		return obj.Name()
	case *types.Func:
		if s.parent != nil {
			return fmt.Sprintf("%v.%v()", s.parent.fieldPath(s.strct), obj.Name())
		}
		return obj.Name()
	default:
		panic(fmt.Sprintf("not implemented for %#v", obj))
	}
}

func (s srcStructEntry) rhsExpr(obj types.Object) string {
	if obj == nil {
		if s.parent != nil {
			return s.parent.rhsExpr(s.strct)
		}
		return s.Name
	}

	switch obj.(type) {
	case *types.Var:
		if s.parent != nil {
			return fmt.Sprintf("%v.%v", s.parent.rhsExpr(s.strct), obj.Name())
		}
		return fmt.Sprintf("%v.%v", s.Name, obj.Name())
	case *types.Func:
		if s.parent != nil {
			return fmt.Sprintf("%v.%v()", s.parent.rhsExpr(s.strct), obj.Name())
		}
		return fmt.Sprintf("%v.%v()", s.Name, obj.Name())
	default:
		panic(fmt.Sprintf("not implemented for %#v", obj))
	}
}

func (s srcStructEntry) isRecursive(newChild *types.Var) bool {
	if s.strct.Id() == newChild.Id() {
		return true
	}
	if s.parent == nil {
		return false
	}
	return s.parent.isRecursive(newChild)
}

type assignmentBuilder struct {
	p           *FunctionBuilder
	methodPos   token.Pos
	opts        option.Options
	assignments []*model.Assignment
}

func newAssignmentBuilder(p *FunctionBuilder, methodPos token.Pos, opts option.Options) *assignmentBuilder {
	return &assignmentBuilder{
		p:           p,
		methodPos:   methodPos,
		opts:        opts,
		assignments: make([]*model.Assignment, 0),
	}
}

func (b *assignmentBuilder) build(srcVar model.Var, src *types.Var, dstVar model.Var, dst types.Type) ([]*model.Assignment, error) {
	srcStrct := srcStructEntry{
		Var:   srcVar,
		strct: src,
	}

	err := util.IterateFields(dst, func(t *types.Var) (done bool, err error) {
		dstField := dstFieldEntry{
			Var:   dstVar,
			field: t,
		}
		a, err := b.create(srcStrct, dstField)
		if err == util.ErrNotFound {
			return false, nil
		}
		if err != nil {
			return
		}
		b.assignments = append(b.assignments, a)
		return
	})
	return b.assignments, err
}

func (b *assignmentBuilder) buildNested(srcParent srcStructEntry, srcChild *types.Var, dstParent dstFieldEntry) (bool, error) {
	if srcParent.isRecursive(srcChild) {
		return false, nil
	}

	srcStrct := srcStructEntry{
		parent: &srcParent,
		Var: model.Var{
			Name:    srcChild.Name(),
			PkgName: srcChild.Pkg().Name(),
			Type:    srcChild.Type().String(),
			Pointer: false,
		},
		strct: srcChild,
	}

	handled := false
	err := util.IterateFields(dstParent.field.Type(), func(t *types.Var) (done bool, err error) {
		dstField := dstFieldEntry{
			parent: &dstParent,
			Var: model.Var{
				Name:    t.Name(),
				PkgName: t.Pkg().Name(),
				Type:    t.Type().String(),
				Pointer: false,
			},
			field: t,
		}
		a, err := b.create(srcStrct, dstField)
		if err == util.ErrNotFound {
			return false, nil
		}
		if err != nil {
			return
		}
		b.assignments = append(b.assignments, a)
		handled = true
		return
	})
	return handled, err
}

func (b *assignmentBuilder) create(src srcStructEntry, dst dstFieldEntry) (*model.Assignment, error) {
	lhs := dst.lhsExpr()

	if !dst.isFieldExported() {
		logger.Printf("%v: skip %v while it is not an exported field", b.p.fset.Position(b.methodPos), lhs)
		return nil, util.ErrNotFound
	}

	if b.opts.ShouldSkip(lhs) {
		logger.Printf("%v: skip %v", b.p.fset.Position(b.methodPos), lhs)
		return &model.Assignment{LHS: lhs, RHS: model.SkipField{}}, nil
	}

	for _, converter := range b.opts.Converters {
		if converter.Dst().Match(dst.fieldName(), true) {
			// If there are more than one converter exist for the dst, the first one wins.
			return b.createWithConverter(src, dst, converter)
		}
	}

	for _, mapper := range b.opts.NameMapper {
		if mapper.Dst().Match(dst.fieldPath(), true) {
			// If there are more than one mapper exist for the dst, the first one wins.
			a, err := b.createWithMapper(src, dst, mapper)
			if err == util.ErrNotFound {
				return &model.Assignment{LHS: lhs, RHS: model.NoMatchField{}}, nil
			}
			return a, err
		}
	}

	return b.createCommon(src, dst)
}

func (b *assignmentBuilder) buildRHS(rhs string, srcType, dstType types.Type) (string, bool) {
	if types.AssignableTo(srcType, dstType) {
		return rhs, true
	}

	if b.opts.Stringer && supportsStringer(srcType, dstType) {
		return rhs + ".String()", true
	}

	if b.opts.Typecast && types.ConvertibleTo(srcType, dstType) {
		if result, ok := b.typeCast(dstType, rhs, b.methodPos); ok {
			return result, true
		}
	}
	return "", false
}

func (b *assignmentBuilder) createCommon(src srcStructEntry, dst dstFieldEntry) (*model.Assignment, error) {
	p := b.p
	opts := b.opts
	methodPos := b.methodPos
	lhs := dst.lhsExpr()

	logger.Printf("%v: lookup assignment for %v = %v.*", p.fset.Position(methodPos), lhs, src.rhsExpr(nil))

	var a *model.Assignment
	var err error
	// To prevent logging "no assignment for d.NestedData"…
	nested := false

	err = util.IterateMethods(src.strct.Type(), func(m *types.Func) (done bool, err error) {
		if src.IsPkgExternal() && !ast.IsExported(m.Name()) {
			return
		}

		retTypes, ok := util.GetMethodReturnTypes(m)
		if !ok || !compliesGetter(retTypes, false) {
			return
		}

		if !opts.Getter || !opts.CompareFieldName(dst.fieldName(), m.Name()) {
			return
		}

		retType := retTypes.At(0).Type()
		returnsError := retTypes.Len() == 2 && util.IsErrorType(retTypes.At(1).Type())
		if rhs, ok := b.buildRHS(src.rhsExpr(m), retType, dst.fieldType()); ok {
			logger.Printf("%v: assignment found: %v = %v", p.fset.Position(methodPos), lhs, rhs)
			a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs, Error: returnsError}}
			return true, nil
		}

		return
	})
	if a != nil || err != nil {
		return a, err
	}

	err = util.IterateFields(src.strctType(), func(f *types.Var) (done bool, err error) {
		if src.IsPkgExternal() && !ast.IsExported(f.Name()) {
			return
		}

		if opts.Rule != model.MatchRuleName || !opts.CompareFieldName(dst.fieldName(), f.Name()) {
			return
		}

		if rhs, ok := b.buildRHS(src.rhsExpr(f), f.Type(), dst.fieldType()); ok {
			logger.Printf("%v: assignment found: %v = %v", p.fset.Position(methodPos), lhs, rhs)
			a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs}}
			return true, nil
		}

		if util.IsStructType(dst.fieldType()) && util.IsStructType(f.Type()) {
			nested = true
			handled, err := b.buildNested(src, f, dst)
			if handled {
				return true, util.ErrNotFound
			}
			return false, err
		}
		return
	})
	if a != nil || err != nil {
		return a, err
	}

	if nested {
		return nil, util.ErrNotFound
	}
	logger.Printf("%v: no assignment for %v [%v]", p.fset.Position(methodPos), lhs, b.p.imports.TypeName(dst.fieldType()))
	return &model.Assignment{LHS: lhs, RHS: model.NoMatchField{}}, nil
}

func (b *assignmentBuilder) createWithConverter(src srcStructEntry, dst dstFieldEntry, converter *option.FieldConverter) (*model.Assignment, error) {
	p := b.p
	opts := b.opts
	methodPos := b.methodPos
	lhs := dst.lhsExpr()

	buildRHSWithConverter := func(srcObj types.Object, srcType types.Type) (string, bool) {
		arg, ok := b.buildRHS(src.rhsExpr(srcObj), srcType, converter.ArgType())
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
			logger.Printf("%v: assignment found: %v = %v", p.fset.Position(methodPos), lhs, rhs)
			a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs, Error: converter.ReturnsError()}}
			return true, nil
		}
		return
	})
	if a != nil || err != nil {
		return a, err
	}

	err = util.IterateFields(src.strctType(), func(f *types.Var) (done bool, err error) {
		if src.IsPkgExternal() && !ast.IsExported(f.Name()) {
			return
		}

		if !converter.Src().Match(f.Name(), true) {
			return
		}
		if rhs, ok := buildRHSWithConverter(f, f.Type()); ok {
			logger.Printf("%v: assignment found: %v = %v", p.fset.Position(methodPos), lhs, rhs)
			a = &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs, Error: converter.ReturnsError()}}
			return true, nil
		}
		return
	})
	if a != nil || err != nil {
		return a, err
	}

	logger.Printf("%v: no assignment for %v [%v]", p.fset.Position(methodPos), lhs, b.p.imports.TypeName(dst.field.Type()))
	return &model.Assignment{LHS: lhs, RHS: model.NoMatchField{}}, nil
}

func (b *assignmentBuilder) createWithMapper(src srcStructEntry, dst dstFieldEntry, mapper *option.NameMatcher) (*model.Assignment, error) {
	p := b.p
	lhs := dst.lhsExpr()
	pos := mapper.Pos()

	expr, obj, ok := b.resolveExpr(mapper.Src(), src.root().strct)
	if !ok {
		logger.Printf("%v: no assignment for %v [%v]", p.fset.Position(pos), lhs, b.p.imports.TypeName(dst.field.Type()))
		return nil, util.ErrNotFound
	}

	rhsExpr := fmt.Sprintf("%v.%v", src.root().Name, expr)

	switch typ := obj.(type) {
	case *types.Func:
		ret, returnsError, ok := util.ParseGetterReturnTypes(typ)
		if !ok {
			logger.Printf("%v: return value mismatch: %v = %v.%v", p.fset.Position(pos), lhs, src.Name, expr)
			return nil, util.ErrNotFound
		}
		rhs, ok := b.buildRHS(rhsExpr, ret, dst.fieldType())
		if !ok {
			logger.Printf("%v: return value mismatch: %v = %v.%v", p.fset.Position(pos), lhs, src.Name, expr)
			return nil, util.ErrNotFound
		}
		logger.Printf("%v: assignment found: %v = %v", p.fset.Position(pos), lhs, rhs)
		return &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs, Error: returnsError}}, nil
	case *types.Var:
		rhs, ok := b.buildRHS(rhsExpr, typ.Type(), dst.fieldType())
		if !ok {
			logger.Printf("%v: return value mismatch: %v = %v.%v", p.fset.Position(pos), lhs, src.Name, expr)
			return nil, util.ErrNotFound
		}
		logger.Printf("%v: assignment found: %v = %v", p.fset.Position(pos), lhs, rhs)
		return &model.Assignment{LHS: lhs, RHS: model.SimpleField{Path: rhs}}, nil
	}

	logger.Printf("%v: no assignment for %v [%v]", p.fset.Position(pos), lhs, b.p.imports.TypeName(dst.field.Type()))
	return nil, util.ErrNotFound
}

func (b *assignmentBuilder) typeCast(t types.Type, inner string, pos token.Pos) (string, bool) {
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
			b.p.fset.Position(pos), b.p.imports.TypeName(t), inner)
		return "", false
	}
}

func (b *assignmentBuilder) isFieldOrMethodAccessible(rcv types.Object, fieldOrMethod types.Object) bool {
	return !b.isExternalPkg(rcv.Pkg()) || ast.IsExported(fieldOrMethod.Name())
}

func (b *assignmentBuilder) isExternalPkg(pkg *types.Package) bool {
	if pkg == nil {
		return false
	}
	return b.p.pkg.PkgPath != pkg.Path()
}

func (b *assignmentBuilder) resolveExpr(matcher *option.IdentMatcher, strct *types.Var) (expr string, obj types.Object, ok bool) {
	var names []string
	typ := strct.Type()

	for i := 0; i < matcher.PathLen(); i++ {
		isLast := matcher.PathLen() == i+1
		pkg := util.PkgOf(typ)

		obj, _, _ = types.LookupFieldOrMethod(typ, false, pkg, matcher.NameAt(i))
		if obj == nil {
			return
		}

		external := b.isExternalPkg(pkg)
		if matcher.ForGetter(i) {
			method, valid := obj.(*types.Func)
			if !valid {
				return
			}
			if external && !ast.IsExported(method.Name()) {
				return
			}

			ret, returnsError, valid := util.ParseGetterReturnTypes(method)
			if !valid {
				return
			}

			names = append(names, method.Name()+"()")
			if isLast {
				expr = strings.Join(names, ".")
				ok = true
				return
			} else {
				if returnsError {
					// It should be a simple getter, otherwise it cannot be a part of method chain.
					return
				}
				typ = ret
			}
		} else {
			field, valid := obj.(*types.Var)
			if !valid {
				return
			}
			if external && !ast.IsExported(field.Name()) {
				return
			}

			names = append(names, field.Name())
			if isLast {
				expr = strings.Join(names, ".")
				ok = true
				return
			} else {
				typ = field.Type()
			}
		}
	}
	return
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
