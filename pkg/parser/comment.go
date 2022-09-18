package parser

import (
	"fmt"
	"go/ast"
	"regexp"
	"strings"

	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/model"
	"github.com/reedom/convergen/pkg/parser/option"
)

var reNotation = regexp.MustCompile(`^\s*//\s*:(\S+)\s*(.*)$`)

type options struct {
	Style model.DstVarStyle
	rule  model.MatchRule

	exactCase bool
	getter    bool
	stringer  bool
	typecast  bool

	receiver    string
	skipFields  []*option.IdentMatcher
	mapFields   []*option.FieldMatcher
	converters  []converter
	postProcess string
}

func newOptions() options {
	return options{
		Style:     model.DstVarReturn,
		rule:      model.MatchRuleName,
		exactCase: true,
		getter:    true,
		stringer:  false,
		typecast:  false,
	}
}

func (o options) copyForMethods() options {
	newOpt := o
	newOpt.receiver = ""
	return newOpt
}

type converter interface {
}

var validOpsIntf = map[string]struct{}{
	"style":        {},
	"match":        {},
	"case":         {},
	"case:off":     {},
	"getter":       {},
	"getter:off":   {},
	"stringer":     {},
	"stringer:off": {},
	"typecast":     {},
	"typecast:off": {},
}

var validOpsMethod = map[string]struct{}{
	"style":        {},
	"match":        {},
	"case":         {},
	"case:off":     {},
	"getter":       {},
	"getter:off":   {},
	"stringer":     {},
	"stringer:off": {},
	"typecast":     {},
	"typecast:off": {},
	"rcv":          {},
	"skip":         {},
	"map":          {},
	"tag":          {},
	"conv":         {},
	"conv:type":    {},
	"conv:with":    {},
	"postprocess":  {},
}

func (p *Parser) parseNotationInComments(notations []*ast.Comment, validOps map[string]struct{}, opts *options) error {
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
				opts.rule = rule
			}
		case "case":
			opts.exactCase = true
		case "case:off":
			opts.exactCase = false
		case "getter":
			opts.getter = true
		case "getter:off":
			opts.getter = false
		case "stringer":
			opts.stringer = true
		case "stringer:off":
			opts.stringer = false
		case "typecast":
			opts.typecast = true
		case "typecast:off":
			opts.typecast = false
		case "rcv":
			if args == nil {
				return logger.Errorf("%v: needs name for the receiver", p.fset.Position(n.Pos()))
			} else if !isValidIdentifier(args[0]) {
				return logger.Errorf("%v: invalid ident", p.fset.Position(n.Pos()))
			}
			opts.receiver = args[0]
		case "skip":
			if args == nil {
				return logger.Errorf("%v: needs <field> arg", p.fset.Position(n.Pos()))
			}
			matcher, err := option.NewIdentMatcher(args[0], opts.exactCase)
			if err != nil {
				return logger.Errorf("%v: invalid <field> arg", p.fset.Position(n.Pos()))
			}
			opts.skipFields = append(opts.skipFields, matcher)
		case "map":
			if len(args) < 2 {
				return logger.Errorf("%v: needs <src> <dst> args", p.fset.Position(n.Pos()))
			}
			matcher, err := option.NewFieldMatcher(args[0], args[1], opts.exactCase)
			if err != nil {
				return logger.Errorf("%v: invalid <field> arg", p.fset.Position(n.Pos()))
			}
			opts.mapFields = append(opts.mapFields, matcher)
		case "conv":
			if len(args) < 2 {
				return logger.Errorf("%v: needs <src> <dst> args", p.fset.Position(n.Pos()))
			}
			scope, obj := p.pkg.Types.Scope().LookupParent(args[0], n.Pos())
			fmt.Printf("@@@ lookup %v, %#v, %#v\n", args[0], scope, obj)
			obj = p.pkg.Types.Scope().Lookup(args[0])
			fmt.Printf("@@@ lookup %v, %#v\n", args[0], obj)
			inner := p.pkg.Types.Scope().Innermost(n.Pos())
			scope, obj = inner.LookupParent("domain", n.Pos())
			fmt.Printf("@@@! lookup %v, %#v, %#v\n", args[0], scope, obj)
		default:
			fmt.Printf("@@@ notation %v\n", m[1])
		}
	}
	return nil
}