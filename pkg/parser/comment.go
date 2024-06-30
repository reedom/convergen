package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"strings"

	bmodel "github.com/reedom/convergen/pkg/builder/model"
	gmodel "github.com/reedom/convergen/pkg/generator/model"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
)

var (
	// reNotation is a regular expression that matches a notation.
	reNotation = regexp.MustCompile(`^\s*//\s*:(\S+)\s*(.*)$`)
	// reConvergen is a regular expression that matches a notation that
	// indicates the beginning of a convergen block.
	reConvergen = regexp.MustCompile(`^\s*//\s*:convergen\b`)
	// reLiteral is a regular expression that matches a notation that
	// indicates the beginning of a literal block.
	reLiteral = regexp.MustCompile(`^\s*\S+\s+(.*)$`)
)

// parseNotationInComments parses given notations and set the values into given Options.
// validOps is a map of valid operation names.
func (p *Parser) parseNotationInComments(notations []*ast.Comment, validOps map[string]struct{}, opts *option.Options) error {
	var posReverse token.Pos

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
		case "convergen":
			// do nothing
		case "style":
			if len(args) == 0 {
				return logger.Errorf("%v: needs <style> arg", p.fset.Position(n.Pos()))
			} else if style, ok := gmodel.NewDstVarStyleFromValue(args[0]); !ok {
				return logger.Errorf("%v: invalid <style> arg", p.fset.Position(n.Pos()))
			} else {
				opts.Style = style
			}
		case "match":
			if len(args) == 0 {
				return logger.Errorf("%v: needs <algorithm> arg", p.fset.Position(n.Pos()))
			} else if rule, ok := gmodel.NewMatchRuleFromValue(args[0]); !ok {
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
		case "recv":
			if len(args) == 0 {
				return logger.Errorf("%v: needs name for the receiver", p.fset.Position(n.Pos()))
			} else if !isValidIdentifier(args[0]) {
				return logger.Errorf("%v: invalid ident", p.fset.Position(n.Pos()))
			}
			opts.Receiver = args[0]

			if len(args) > 1 { // with function prefix to cut
				if !isValidIdentifier(args[1]) {
					return logger.Errorf("%v: invalid ident", p.fset.Position(n.Pos()))
				}

				opts.FuncCutPrefix = args[1]
			}
		case "reverse":
			opts.Reverse = true
			posReverse = n.Pos()
		case "skip":
			if len(args) == 0 {
				return logger.Errorf("%v: needs <field> arg", p.fset.Position(n.Pos()))
			}
			matcher, err := option.NewPatternMatcher(args[0], opts.ExactCase)
			if err != nil {
				return logger.Errorf("%v: invalid regexp", p.fset.Position(n.Pos()))
			}
			opts.SkipFields = append(opts.SkipFields, matcher)
		case "map":
			if len(args) < 2 {
				return logger.Errorf("%v: needs <src> <dst>", p.fset.Position(n.Pos()))
			}
			matcher := option.NewNameMatcher(args[0], args[1], n.Pos())
			opts.NameMapper = append(opts.NameMapper, matcher)
		case "conv":
			if len(args) < 2 {
				return logger.Errorf("%v: needs <conv> <src> <dst>", p.fset.Position(n.Pos()))
			}
			src := args[1]
			dst := src
			if 3 <= len(args) {
				dst = args[2]
			}
			converter := option.NewFieldConverter(args[0], src, dst, n.Pos())
			opts.Converters = append(opts.Converters, converter)
		case "method", "method:err":
			if len(args) < 2 {
				return logger.Errorf("%v: needs <method> <src> <dst>", p.fset.Position(n.Pos()))
			}

			src := args[1]
			dst := src
			if 3 <= len(args) {
				dst = args[2]
			}

			method := option.NewFieldConverter(args[0], src, dst, n.Pos())
			if m[1] == "method:err" {
				// ! method can trim prefix, can NOT calc by programe
				method.Set(nil, nil, true) // force error return
			}

			opts.Methods = append(opts.Methods, method)
		case "literal":
			if len(args) < 2 {
				return logger.Errorf("%v: needs <dst> <literal> args", p.fset.Position(n.Pos()))
			}
			m = reLiteral.FindStringSubmatch(m[2])
			setter := option.NewLiteralSetter(args[0], m[1], n.Pos())
			opts.Literals = append(opts.Literals, setter)
		case "preprocess":
			if len(args) < 1 {
				return logger.Errorf("%v: needs <func> arg", p.fset.Position(n.Pos()))
			}
			pp, err := p.lookupManipulatorFunc(args[0], "preprocess", n.Pos())
			if err != nil {
				return err
			}
			opts.PreProcess = pp
		case "postprocess":
			if len(args) < 1 {
				return logger.Errorf("%v: needs <func> arg", p.fset.Position(n.Pos()))
			}
			pp, err := p.lookupManipulatorFunc(args[0], "postprocess", n.Pos())
			if err != nil {
				return err
			}
			opts.PostProcess = pp
		default:
			fmt.Printf("%v: unknown notation %v\n", p.fset.Position(n.Pos()), m[1])
		}
	}

	// validation
	if opts.Reverse && opts.Style == gmodel.DstVarReturn {
		return logger.Errorf(`%v: to use ":reverse", style must be ":style arg"`, p.fset.Position(posReverse))
	}

	return nil
}

// lookupType looks up a type by name in the current package or its imports.
// It returns the scope and object of the type if found, or nil if not found.
// typeName is the fully qualified name of the type, including package name.
// pos is the position where the lookup occurs.
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

// resolveConverters resolves the types and error flag of the FieldConverter `conv` by
// looking up the corresponding function based on the converter's name. If the function
// is found, its argument and return types are set to `conv`. If not, it tries to find
// a method in `generatingMethods` that can be used as a converter.
//
// If no function or method is found, an error is returned.
func (p *Parser) resolveConverters(generatingMethods []*bmodel.MethodEntry, conv *option.FieldConverter) error {
	name := conv.Converter()
	pos := conv.Pos()
	argType, retType, retError, err := p.lookupConverterFunc(name, pos)
	if err == nil {
		conv.Set(argType, retType, retError)
		return nil
	}

	for _, method := range generatingMethods {
		if method.Name() != name {
			continue
		}
		if method.Opts.Style != gmodel.DstVarReturn {
			err = logger.Errorf("%v: function %v cannot use as a converter", p.fset.Position(pos), name)
			continue
		}
		if method.Recv() != nil {
			// TODO(reedom): we may accept a method as a converter.
			err = logger.Errorf("%v: function %v cannot use as a converter", p.fset.Position(pos), name)
			continue
		}
		conv.Set(method.SrcVar().Type(), method.DstVar().Type(), method.RetError())
		return nil
	}

	if err == nil {
		err = logger.Errorf("%v: function %v not found", p.fset.Position(pos), name)
	}
	return err
}

// lookupConverterFunc finds and returns the argument and return types of a function
// with the given name and position.
// It checks that the function is a valid converter function and can be used as such.
func (p *Parser) lookupConverterFunc(funcName string, pos token.Pos) (argType, retType types.Type, retError bool, err error) {
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
	retError = sig.Results().Len() == 2 && util.IsErrorType(sig.Results().At(1).Type())
	return
}

// lookupManipulatorFunc looks up a function by name and verifies that it can be used
// as a manipulator function for a certain option. It returns a new Manipulator instance
// on success, and an error on failure.
func (p *Parser) lookupManipulatorFunc(funcName, optName string, pos token.Pos) (*option.Manipulator, error) {
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
		return nil, logger.Errorf("%v: function %v cannot use for %v func", p.fset.Position(pos), funcName, optName)
	}

	return &option.Manipulator{
		Func:     obj,
		DstSide:  sig.Params().At(0).Type(),
		SrcSide:  sig.Params().At(1).Type(),
		RetError: sig.Results().Len() == 1 && util.IsErrorType(sig.Results().At(0).Type()),
		Pos:      pos,
	}, nil
}
