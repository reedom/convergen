package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"

	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/model"
	"golang.org/x/tools/go/loader"
)

func (p *Parser) createAssign(opts options, dst *types.Var, dstVar model.Var, srcType types.Type, srcVar model.Var, pos token.Pos) (*model.Assignment, error) {
	name := dst.Name()
	dstVarName := fmt.Sprintf("%v.%v", dstVar.Name, name)

	if dstVar.PkgName != "" && !ast.IsExported(name) {
		logger.Printf("%v: skip %v.%v while it is not an exported field",
			p.fset.Position(dst.Pos()), dstVar.Name, dst.Name())
		return nil, errNotFound
	}

	// :skip notation
	if opts.shouldSkip(dstVarName) {
		logger.Printf("%v: skip %v.%v [%v]",
			p.fset.Position(dst.Pos()), dstVar.Name, dst.Name(), dst.Type().String())
		return &model.Assignment{
			LHS: dstVarName,
			RHS: model.SkipField{},
		}, nil
	}

	// Handle getters
	if opts.getter {
		a, err := p.createAssignFromGetters(opts, dst, dstVar, srcType, srcVar)
		if err != nil && err != errNotFound {
			return nil, err
		}
		if a != nil {
			return a, nil
		}
	}

	// Field name mapping
	if opts.rule == model.MatchRuleName {
		a, err := p.createAssignFromFields(opts, dst, dstVar, srcType, srcVar)
		if err != nil && err != errNotFound {
			return nil, err
		}
		if a != nil {
			return a, nil
		}
	}

	logger.Printf("%v: no assignment for %v.%v [%v]",
		p.fset.Position(dst.Pos()), dstVar.Name, dst.Name(), dst.Type().String())
	return &model.Assignment{
		LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
		RHS: model.NoMatchField{},
	}, nil
}

func (p *Parser) createAssignFromFields(opts options, dst *types.Var, dstVar model.Var, srcType types.Type, srcVar model.Var) (*model.Assignment, error) {
	name := dst.Name()
	var a *model.Assignment

	err := iterateFields(srcType, func(f *types.Var) (done bool, err error) {
		if !opts.compareFieldName(name, f.Name()) {
			return
		}
		if srcVar.IsPkgExternal() && !ast.IsExported(f.Name()) {
			return
		}

		if types.AssignableTo(f.Type(), dst.Type()) {
			logger.Printf("%v: assignment found, %v.%v [%v] to %v.%v [%v]",
				p.fset.Position(dst.Pos()), srcVar.Name, f.Name(), f.Type().String(),
				dstVar.Name, dst.Name(), dst.Type().String())
			a = &model.Assignment{
				LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
				RHS: model.SimpleField{Path: fmt.Sprintf("%v.%v", srcVar.Name, f.Name())},
			}
			return true, nil
		}

		// :stringer notation
		if opts.stringer && supportsStringer(f.Type(), dst.Type()) {
			logger.Printf("%v: assignment found, %v.%v.String() to %v.%v [%v]",
				p.fset.Position(dst.Pos()), srcVar.Name, f.Name(), f.Type().String(),
				dstVar.Name, dst.Name(), dst.Type().String())
			a = &model.Assignment{
				LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
				RHS: model.SimpleField{Path: fmt.Sprintf("%v.%v.String()", srcVar.Name, f.Name())},
			}
			return true, nil
		}

		// :typecast notation
		if opts.typecast && types.ConvertibleTo(f.Type(), dst.Type()) {
			logger.Printf("%v: assignment found, %v(%v.%v) to %v.%v",
				p.fset.Position(dst.Pos()), dst.Type().String(), srcVar.Name, f.Name(), f.Type().String(),
				dstVar.Name, dst.Name())
			a = &model.Assignment{
				LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
				RHS: model.SimpleField{Path: fmt.Sprintf("%v(%v.%v)", dst.Type().String(), srcVar.Name, f.Name())},
			}
			return true, nil
		}
		return
	})

	return a, err
}

func (p *Parser) createAssignFromGetters(opts options, dst *types.Var, dstVar model.Var, srcType types.Type, srcVar model.Var) (*model.Assignment, error) {
	name := dst.Name()
	var a *model.Assignment

	err := iterateMethods(srcType, func(m *types.Func) (done bool, err error) {
		if !opts.compareFieldName(name, m.Name()) {
			return
		}
		if srcVar.IsPkgExternal() && !ast.IsExported(m.Name()) {
			return
		}

		retTypes, ok := getMethodReturnTypes(m)
		if !ok || !compliesGetter(retTypes, false) {
			return
		}

		retType := retTypes.At(0).Type()
		if types.AssignableTo(retType, dst.Type()) {
			logger.Printf("%v: assignment found, %v.%v() [%v] to %v.%v [%v]",
				p.fset.Position(dst.Pos()), srcVar.Name, m.Name(), m.Type().Underlying().String(),
				dstVar.Name, dst.Name(), dst.Type().String())
			a = &model.Assignment{
				LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
				RHS: model.SimpleField{Path: fmt.Sprintf("%v.%v()", srcVar.Name, m.Name())},
			}
			return true, nil
		}

		// :stringer notation
		if opts.stringer && supportsStringer(retType, dst.Type()) {
			logger.Printf("%v: assignment found, %v.%v().String() to %v.%v [%v]",
				p.fset.Position(dst.Pos()), srcVar.Name, m.Name(),
				dstVar.Name, dst.Name(), dst.Type().String())
			a = &model.Assignment{
				LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
				RHS: model.SimpleField{Path: fmt.Sprintf("%v.%v().String()", srcVar.Name, m.Name())},
			}
			return true, nil
		}

		// :typecast notation
		if opts.typecast && types.ConvertibleTo(retType, dst.Type()) {
			logger.Printf("%v: assignment found, %v(%v.%v) to %v.%v",
				p.fset.Position(dst.Pos()), dst.Type().String(), srcVar.Name, dstVar.Name, dst.Name(),
				dstVar.Name, dst.Name())
			a = &model.Assignment{
				LHS: fmt.Sprintf("%v.%v", dstVar.Name, name),
				RHS: model.SimpleField{Path: fmt.Sprintf("%v(%v.%v)", dst.Type().String(), srcVar.Name, m.Name())},
			}
			return true, nil
		}

		return
	})
	return a, err
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
