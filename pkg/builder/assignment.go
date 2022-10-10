package builder

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	bmodel "github.com/reedom/convergen/pkg/builder/model"
	gmodel "github.com/reedom/convergen/pkg/generator/model"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
	"golang.org/x/tools/go/packages"
)

type assignmentBuilder struct {
	file    *ast.File
	fset    *token.FileSet
	pkg     *packages.Package
	imports util.ImportNames

	methodPos token.Pos
	opts      option.Options
	lhsVar    gmodel.Var
	rhsVar    gmodel.Var

	funcName string
	copiers  []*bmodel.Copier
}

func newAssignmentBuilder(p *FunctionBuilder, m *bmodel.MethodEntry, lhsVar, rhsVar gmodel.Var) *assignmentBuilder {
	return &assignmentBuilder{
		file:      p.file,
		fset:      p.fset,
		pkg:       p.pkg,
		imports:   p.imports,
		methodPos: m.Method.Pos(),
		opts:      m.Opts,
		lhsVar:    lhsVar,
		rhsVar:    rhsVar,
		funcName:  m.Name(),
	}
}

func (b *assignmentBuilder) build(lhs, rhs *types.Var) ([]gmodel.Assignment, error) {
	rootCopier := bmodel.NewCopier("", lhs.Type(), rhs.Type())
	rootCopier.IsRoot = true
	if b.opts.Receiver != "" {
		rootCopier.Name = fmt.Sprintf("%v.%v", b.lhsVar.Name, b.funcName)
	} else {
		rootCopier.Name = b.funcName
	}
	b.copiers = append(b.copiers, rootCopier)

	rootLHS := bmodel.NewRootNode(b.lhsVar.Name, lhs.Type())
	rootRHS := bmodel.NewRootNode(b.rhsVar.Name, rhs.Type())
	return b.dispatch(rootLHS, rootRHS)
}

func (b *assignmentBuilder) dispatch(lhs, rhs bmodel.Node) ([]gmodel.Assignment, error) {
	lhsType := util.DerefPtr(lhs.ExprType())
	rhsType := util.DerefPtr(rhs.ExprType())
	if util.IsStructType(lhsType) && util.IsStructType(rhsType) {
		return b.structToStruct(lhs, rhs)
	}

	logger.Warnf("%v: no assignment %T to %T", b.fset.Position(b.methodPos), rhs.ExprType(), lhs.ExprType())
	return []gmodel.Assignment{gmodel.NoMatchField{LHS: lhs.AssignExpr()}}, nil
}

func (b *assignmentBuilder) structToStruct(lhsStruct, rhsStruct bmodel.Node) ([]gmodel.Assignment, error) {
	var err error
	var assignments []gmodel.Assignment
	bmodel.IterateStructFields(lhsStruct, func(lhsField bmodel.Node) (done bool) {
		if !b.isStructFieldAccessible(lhsStruct, lhsField.ObjName()) {
			return
		}

		var a gmodel.Assignment
		a, err = b.matchStructFieldAndStruct(lhsField, rhsStruct)
		if err == nil && a != nil {
			assignments = append(assignments, a)
		}
		return
	})
	return assignments, err
}

func (b *assignmentBuilder) matchStructFieldAndStruct(lhs bmodel.Node, rhs bmodel.Node) (gmodel.Assignment, error) {
	if b.opts.ShouldSkip(lhs.MatcherExpr()) {
		logger.Printf("%v: skip %v", b.fset.Position(b.methodPos), lhs.AssignExpr())
		return gmodel.SkipField{LHS: lhs.AssignExpr()}, nil
	}

	for _, converter := range b.opts.Converters {
		if converter.Dst().Match(lhs.MatcherExpr(), true) {
			// If there are more than one converter exist for the lhs, the first one wins.
			return b.createWithConverter(lhs, rhs, converter)
		}
	}

	for _, mapper := range b.opts.NameMapper {
		if mapper.Dst().Match(lhs.MatcherExpr(), true) {
			// If there are more than one mapper exist for the lhs, the first one wins.
			return b.createWithMapper(lhs, rhs, mapper)
		}
	}

	return b.structFieldAndStructGettersAndFields(lhs, rhs)
}

func (b *assignmentBuilder) structFieldAndStructGettersAndFields(lhs bmodel.Node, rhsStruct bmodel.Node) (gmodel.Assignment, error) {
	opts := b.opts
	methodPosStr := b.fset.Position(b.methodPos)
	lhsExpr := lhs.AssignExpr()

	logger.Printf("%v: lookup assignment for %v = %v.*", methodPosStr, lhsExpr, rhsStruct.AssignExpr())

	var a gmodel.Assignment
	var err error
	// To prevent logging "no assignment for d.NestedData"…
	nested := false

	handler := func(rhs bmodel.Node) (done bool) {
		if !b.isStructFieldAccessible(rhsStruct, rhs.ObjName()) ||
			!opts.CompareFieldName(lhs.ObjName(), rhs.ObjName()) {
			return
		}

		if util.IsSliceType(lhs.ExprType()) && util.IsSliceType(rhs.ExprType()) {
			a, err = b.sliceToSlice(lhs, rhs)
			if a != nil || err != nil {
				logger.Printf("%v: assignment found: sliceCopy(%v, %v)", methodPosStr, lhsExpr, rhs.AssignExpr())
				return true
			}
		}

		if c, ok := b.castNode(lhs.ExprType(), rhs); ok {
			rhsExpr := c.AssignExpr()
			logger.Printf("%v: assignment found: %v = %v", methodPosStr, lhsExpr, rhsExpr)
			a = gmodel.SimpleField{LHS: lhsExpr, RHS: rhsExpr, Error: c.ReturnsError()}
			return true
		}

		if util.IsStructType(lhs.ExprType()) &&
			util.IsStructType(rhs.ExprType()) {
			nested = true
			nestStruct := gmodel.NestStruct{}
			if util.IsPtr(lhs.ExprType()) {
				nestStruct.InitExpr = fmt.Sprintf("%v = %v{}", lhs.AssignExpr(), b.imports.TypeName(lhs.ExprType()))
			}
			if rhs.ObjNullable() {
				nestStruct.NullCheckExpr = rhs.NullCheckExpr()
			}
			nestStruct.Contents, err = b.structToStruct(lhs, rhs)
			if err == nil && 0 < len(nestStruct.Contents) {
				a = nestStruct
			}
		}
		return true
	}

	if opts.Getter {
		bmodel.IterateStructMethods(rhsStruct, handler)
		if a != nil || err != nil {
			return a, err
		}
	}

	if opts.Rule == gmodel.MatchRuleName {
		bmodel.IterateStructFields(rhsStruct, handler)
		if a != nil || err != nil || nested {
			return a, err
		}
	}

	logger.Warnf("%v: no assignment for %v [%v]", methodPosStr, lhsExpr, b.imports.TypeName(lhs.ExprType()))
	return gmodel.NoMatchField{LHS: lhsExpr}, nil
}

func (b *assignmentBuilder) createWithConverter(lhs, rhs bmodel.Node, converter *option.FieldConverter) (gmodel.Assignment, error) {
	converterNode := func() bmodel.Node {
		root := rhs
		for ; root.Parent() != nil; root = root.Parent() {
		}

		rhsNode, ok := b.resolveExpr(converter.Src(), root)
		if !ok {
			return nil
		}

		argNode, ok := b.castNode(converter.ArgType(), rhsNode)
		if !ok {
			if !util.IsPtr(converter.ArgType()) {
				return nil
			}
			argNode, ok = b.castNode(util.DerefPtr(converter.ArgType()), rhsNode)
			if !ok {
				return nil
			}
		}
		convNode := bmodel.NewConverterNode(argNode, converter)
		casted, _ := b.castNode(lhs.ExprType(), convNode)
		return casted
	}()

	lhsExpr := lhs.AssignExpr()
	posStr := b.fset.Position(converter.Pos())

	if converterNode != nil {
		rhsExpr := converterNode.AssignExpr()
		logger.Printf("%v: assignment found: %v = %v, err", posStr, lhsExpr, rhsExpr)
		return gmodel.SimpleField{LHS: lhsExpr, RHS: rhsExpr, Error: converter.RetError()}, nil
	}

	logger.Warnf("%v: no assignment for %v [%v]", posStr, lhsExpr, b.imports.TypeName(lhs.ExprType()))
	return gmodel.NoMatchField{LHS: lhsExpr}, nil
}

func (b *assignmentBuilder) createWithMapper(lhs, rhs bmodel.Node, mapper *option.NameMatcher) (gmodel.Assignment, error) {
	mappedNode := func() bmodel.Node {
		root := rhs
		for ; root.Parent() != nil; root = root.Parent() {
		}

		rhsNode, ok := b.resolveExpr(mapper.Src(), root)
		if !ok {
			return nil
		}

		casted, _ := b.castNode(lhs.ExprType(), rhsNode)
		return casted
	}()

	lhsExpr := lhs.AssignExpr()
	posStr := b.fset.Position(mapper.Pos())

	if mappedNode != nil {
		rhsExpr := mappedNode.AssignExpr()
		logger.Printf("%v: assignment found: %v = %v", posStr, lhs, rhs)
		return gmodel.SimpleField{LHS: lhsExpr, RHS: rhsExpr, Error: mappedNode.ReturnsError()}, nil
	}

	logger.Warnf("%v: no assignment for %v [%v]", posStr, lhsExpr, b.imports.TypeName(lhs.ExprType()))
	return gmodel.NoMatchField{LHS: lhsExpr}, nil
}

func (b *assignmentBuilder) castNode(lhsType types.Type, rhs bmodel.Node) (c bmodel.Node, ok bool) {
	if types.AssignableTo(rhs.ExprType(), lhsType) {
		return rhs, true
	}

	if b.opts.Stringer && types.AssignableTo(util.StringType(), lhsType) && util.CompliesStringer(rhs.ExprType()) {
		return b.castNode(lhsType, bmodel.NewStringer(rhs))
	}

	if b.opts.Typecast && types.ConvertibleTo(rhs.ExprType(), lhsType) {
		c, ok = bmodel.NewTypecast(b.pkg.Types.Scope(), b.imports, lhsType, rhs)
		if !ok {
			logger.Warnf("%v: typecast for %v is not implemented(yet) for %v",
				b.fset.Position(b.methodPos), b.imports.TypeName(lhsType), rhs.AssignExpr())
		}
		return
	}
	return nil, false
}

func (b *assignmentBuilder) isStructFieldAccessible(structNode bmodel.Node, leafName string) bool {
	structType := util.DerefPtr(structNode.ExprType())
	if !util.IsStructType(structType) {
		return false
	}
	if named, ok := structType.(*types.Named); ok {
		return !b.isExternalPkg(named.Obj().Pkg()) || ast.IsExported(leafName)
	}
	return true

}

func (b *assignmentBuilder) isExternalPkg(pkg *types.Package) bool {
	if pkg == nil {
		return false
	}
	return b.pkg.PkgPath != pkg.Path()
}

func (b *assignmentBuilder) resolveExpr(matcher *option.IdentMatcher, root bmodel.Node) (node bmodel.Node, ok bool) {
	node = root
	typ := root.ExprType()
	for i := 0; i < matcher.PathLen(); i++ {
		isLast := matcher.PathLen() == i+1
		pkg := util.PkgOf(typ)

		obj, _, _ := types.LookupFieldOrMethod(typ, false, pkg, matcher.NameAt(i))
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

			ret, retError, valid := util.ParseGetterReturnTypes(method)
			if !valid {
				return
			}

			node = bmodel.NewStructMethodNode(node, method)
			if isLast {
				return node, true
			}

			if retError {
				// It should be a simple getter, otherwise it cannot be a part of method chain.
				return
			}
			typ = ret
		} else {
			field, valid := obj.(*types.Var)
			if !valid {
				return
			}
			if external && !ast.IsExported(field.Name()) {
				return
			}

			node = bmodel.NewStructFieldNode(node, field)
			if isLast {
				return node, true
			}

			typ = field.Type()
		}
	}
	return
}

func (b *assignmentBuilder) sliceToSlice(lhs, rhs bmodel.Node) (a gmodel.Assignment, err error) {
	lhsElem := util.SliceElement(lhs.ExprType())
	rhsElem := util.SliceElement(rhs.ExprType())
	if lhsElem == nil || rhsElem == nil {
		return
	}

	if types.AssignableTo(rhsElem, lhsElem) {
		if util.IsBasicType(rhsElem) {
			a = gmodel.SliceAssignment{
				LHS: lhs.AssignExpr(),
				RHS: rhs.AssignExpr(),
				Typ: "[]" + lhsElem.String(),
			}
		} else {
			a = gmodel.SliceLoopAssignment{
				LHS: lhs.AssignExpr(),
				RHS: rhs.AssignExpr(),
				Typ: "[]" + b.imports.TypeName(lhsElem),
			}
		}
		return
	}

	if b.opts.Typecast && types.ConvertibleTo(rhsElem, lhsElem) {
		a = gmodel.SliceTypecastAssignment{
			LHS:  lhs.AssignExpr(),
			RHS:  rhs.AssignExpr(),
			Typ:  "[]" + b.imports.TypeName(lhsElem),
			Cast: b.imports.TypeName(lhsElem),
		}
		return
	}
	return
}
