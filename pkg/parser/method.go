package parser

import (
	"errors"
	"fmt"
	"go/types"
	"os"
	"regexp"

	"github.com/reedom/convergen/pkg/builder/model"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
)

var (
	// reGoBuildGen is a regular expression that matches a notation that
	// indicates the beginning of a convergen block.
	reGoBuildGen = regexp.MustCompile(`\s*//\s*(go:(generate\b|build convergen\b)|\+build convergen)`)
	// reConvergen is a regular expression that matches a notation that
	// indicates the beginning of a convergen block.
	errAbort = errors.New("abort")
)

// parseMethods parses all the methods in an interface type.
func (p *Parser) parseMethods(intf *intfEntry) ([]*model.MethodEntry, error) {
	iface := intf.intf.Type().Underlying().(*types.Interface)
	mset := types.NewMethodSet(iface)
	methods := make([]*model.MethodEntry, 0)
	for i := 0; i < mset.Len(); i++ {
		method, err := p.parseMethod(mset.At(i).Obj(), intf.opts)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			continue
		}
		methods = append(methods, method)
	}
	if len(methods) < mset.Len() {
		return nil, errAbort
	}

	return methods, nil
}

// parseMethod parses a single method in an interface type.
func (p *Parser) parseMethod(method types.Object, opts option.Options) (*model.MethodEntry, error) {
	signature, ok := method.Type().(*types.Signature)
	if !ok {
		return nil, logger.Errorf(`%v: expected signature but %#v`, p.fset.Position(method.Pos()), method)
	}

	if signature.Params().Len() == 0 {
		return nil, logger.Errorf(`%v: method must have one or more arguments as copy source`, p.fset.Position(method.Pos()))
	}
	if signature.Results().Len() == 0 {
		return nil, logger.Errorf(`%v: method must have one or more return values as copy destination`, p.fset.Position(method.Pos()))
	}

	docComment, cleanUp := util.GetDocCommentOn(p.file, method)
	notations := util.ExtractMatchComments(docComment, reNotation)
	err := p.parseNotationInComments(notations, option.ValidOpsMethod, &opts)
	if err != nil {
		return nil, err
	}

	cleanUp()

	return &model.MethodEntry{
		Method:     method,
		Opts:       opts,
		DocComment: docComment,
	}, nil
}
