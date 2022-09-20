package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"strings"

	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/model"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
)

var reNotation = regexp.MustCompile(`^\s*//\s*:(\S+)\s*(.*)$`)

func (p *Parser) parseNotationInComments(notations []*ast.Comment, validOps map[string]struct{}, opts *option.Options) error {
	for _, n := range notations {
		m := reNotation.FindStringSubmatch(n.Text)
		if m == nil || len(m) < 2 {
			return fmt.Errorf("invalid notation format %#v", m)
		}

		var args []string
		if len(m) == 3 {
			args = strings.Fields(m[2])
		}

		if _, ok := validOps[m[1]]; !ok {
			logger.Printf(`%v: ":%v" is invalid or unknown notation here`, p.fset.Position(n.Pos()), m[1])
			continue
		}

		switch m[1] {
		case "style":
			if args == nil {
				return logger.Errorf("%v: needs <style> arg", p.fset.Position(n.Pos()))
			} else if style, ok := model.NewDstVarStyleFromValue(args[0]); !ok {
				return logger.Errorf("%v: invalid <style> arg", p.fset.Position(n.Pos()))
			} else {
				opts.Style = style
			}
		case "match":
			if args == nil {
				return logger.Errorf("%v: needs <algorithm> arg", p.fset.Position(n.Pos()))
			} else if rule, ok := model.NewMatchRuleFromValue(args[0]); !ok {
				return logger.Errorf("%v: invalid <algorithm> arg", p.fset.Position(n.Pos()))
			} else {
				opts.Rule = rule
			}
		case "case":
			opts.ExactCase = true
		case "case:off":
			opts.ExactCase = false
		case "getter":
			opts.Getter = true
		case "getter:off":
			opts.Getter = false
		case "stringer":
			opts.Stringer = true
		case "stringer:off":
			opts.Stringer = false
		case "typecast":
			opts.Typecast = true
		case "typecast:off":
			opts.Typecast = false
		case "rcv":
			if args == nil {
				return logger.Errorf("%v: needs name for the receiver", p.fset.Position(n.Pos()))
			} else if !isValidIdentifier(args[0]) {
				return logger.Errorf("%v: invalid ident", p.fset.Position(n.Pos()))
			}
			opts.Receiver = args[0]
		case "skip":
			if args == nil {
				return logger.Errorf("%v: needs <field> arg", p.fset.Position(n.Pos()))
			}
			matcher, err := option.NewIdentMatcher(args[0], opts.ExactCase)
			if err != nil {
				return logger.Errorf("%v: invalid <field> arg", p.fset.Position(n.Pos()))
			}
			opts.SkipFields = append(opts.SkipFields, matcher)
		case "map":
			if len(args) < 2 {
				return logger.Errorf("%v: needs <src> <dst> args", p.fset.Position(n.Pos()))
			}
			matcher, err := option.NewNameMatcher(args[0], args[1], true, n.Pos())
			if err != nil {
				return logger.Errorf("%v: %v", p.fset.Position(n.Pos()), err.Error())
			}
			opts.NameMapper = append(opts.NameMapper, matcher)
		case "conv":
			if len(args) < 2 {
				return logger.Errorf("%v: needs <src> <dst> args", p.fset.Position(n.Pos()))
			}
			argType, retType, returnsError, err := p.lookupConverterFunc(args[0], n.Pos())
			if err != nil {
				return err
			}
			src := args[1]
			dst := src
			if 3 <= len(args) {
				dst = args[2]
			}
			converter, err := option.NewFieldConverter(args[0], src, dst,
				argType, retType, returnsError, n.Pos())
			if err != nil {
				return logger.Errorf("%v: %v", p.fset.Position(n.Pos()), err.Error())
			}
			opts.Converters = append(opts.Converters, converter)
		case "postprocess":
			if len(args) < 1 {
				return logger.Errorf("%v: needs <func> arg", p.fset.Position(n.Pos()))
			}
			pp, err := p.lookupPostprocessFunc(args[0], n.Pos())
			if err != nil {
				return err
			}
			opts.PostProcess = pp
		default:
			fmt.Printf("@@@ notation %v\n", m[1])
		}
	}
	return nil
}

func (p *Parser) lookupType(typeName string, pos token.Pos) (*types.Scope, types.Object) {
	names := strings.Split(typeName, ".")
	if len(names) == 1 {
		inner := p.pkg.Types.Scope().Innermost(pos)
		return inner.LookupParent(names[0], pos)
	}

	pkgPath, ok := p.imports.LookupPath(names[0])
	if !ok {
		return nil, nil
	}
	pkg, ok := p.pkg.Imports[pkgPath]
	if !ok {
		return nil, nil
	}

	scope := pkg.Types.Scope()
	obj := scope.Lookup(names[1])
	return scope, obj
}

func (p *Parser) lookupConverterFunc(funcName string, pos token.Pos) (argType, retType types.Type, returnsError bool, err error) {
	_, obj := p.lookupType(funcName, pos)
	if obj == nil {
		err = logger.Errorf("%v: function %v not found", p.fset.Position(pos), funcName)
		return
	}
	sig, ok := obj.Type().(*types.Signature)
	if !ok {
		err = logger.Errorf("%v: %v isn't a function", p.fset.Position(pos), funcName)
		return
	}
	if sig.Params().Len() != 1 || sig.Results().Len() < 1 || 2 < sig.Results().Len() {
		err = logger.Errorf("%v: function %v cannot use as a converter", p.fset.Position(pos), funcName)
		return
	}
	if sig.Results().Len() == 2 && !util.IsErrorType(sig.Results().At(1).Type()) {
		err = logger.Errorf("%v: function %v cannot use as a converter", p.fset.Position(pos), funcName)
		return
	}

	argType = sig.Params().At(0).Type()
	retType = sig.Results().At(0).Type()
	returnsError = sig.Results().Len() == 2 && util.IsErrorType(sig.Results().At(1).Type())
	return
}

func (p *Parser) lookupPostprocessFunc(funcName string, pos token.Pos) (*option.Postprocess, error) {
	_, obj := p.lookupType(funcName, pos)
	if obj == nil {
		return nil, logger.Errorf("%v: function %v not found", p.fset.Position(pos), funcName)
	}
	sig, ok := obj.Type().(*types.Signature)
	if !ok {
		return nil, logger.Errorf("%v: %v isn't a function", p.fset.Position(pos), funcName)
	}

	if sig.Params().Len() != 2 ||
		1 < sig.Results().Len() ||
		(sig.Results().Len() == 1 && !util.IsErrorType(sig.Results().At(0).Type())) {
		return nil, logger.Errorf("%v: function %v cannot use for postprocess func", p.fset.Position(pos), funcName)
	}

	return &option.Postprocess{
		Func:         obj,
		DstSide:      sig.Params().At(0).Type(),
		SrcSide:      sig.Params().At(1).Type(),
		ReturnsError: sig.Results().Len() == 1 && util.IsErrorType(sig.Results().At(0).Type()),
		Pos:          pos,
	}, nil
}
