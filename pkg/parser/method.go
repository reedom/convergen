package parser

import (
	"context"
	"errors"
	"fmt"
	"go/types"
	"os"
	"regexp"

	"go.uber.org/zap"

	"github.com/reedom/convergen/v9/pkg/builder/model"
	"github.com/reedom/convergen/v9/pkg/logger"
	"github.com/reedom/convergen/v9/pkg/option"
	"github.com/reedom/convergen/v9/pkg/util"
)

var (
	// reGoBuildGen is a regular expression that matches a notation that
	// indicates the beginning of a convergen block.
	reGoBuildGen = regexp.MustCompile(`\s*//\s*(go:(generate\b|build convergen\b)|\+build convergen)`)
	// reConvergen is a regular expression that matches a notation that
	// indicates the beginning of a convergen block.
	errAbort = errors.New("abort")
)

// parseMethods parses all the methods in an interface type with optional concurrency.
func (p *Parser) parseMethods(intf *intfEntry) ([]*model.MethodEntry, error) {
	// Use concurrent processing if enabled and we have sufficient methods
	if p.config != nil && p.config.EnableMethodConcurrency {
		return p.parseMethodsConcurrent(intf)
	}

	// Fallback to sequential processing
	return p.parseMethodsSequential(intf)
}

// parseMethodsConcurrent processes methods concurrently for better performance.
func (p *Parser) parseMethodsConcurrent(intf *intfEntry) ([]*model.MethodEntry, error) {
	iface := intf.intf.Type().Underlying().(*types.Interface)
	mset := types.NewMethodSet(iface)

	// Check for empty interfaces - they should be rejected
	if mset.Len() == 0 {
		return nil, fmt.Errorf("%v: interface %s has no methods - expected declaration",
			p.fset.Position(intf.intf.Pos()), intf.intf.Name())
	}

	// Create logger for concurrent processing
	zapLogger := zap.NewNop() // Use nop logger for now, can be enhanced later

	// Create concurrent processor
	processor := NewConcurrentMethodProcessor(
		p,
		p.config.MaxConcurrentWorkers,
		p.config.TypeResolutionTimeout,
		zapLogger,
	)

	// Process methods concurrently
	ctx, cancel := context.WithTimeout(context.Background(), p.config.TypeResolutionTimeout*2)
	defer cancel()

	return processor.ProcessMethodsConcurrent(ctx, intf)
}

// parseMethodsSequential processes methods sequentially (legacy behavior).
func (p *Parser) parseMethodsSequential(intf *intfEntry) ([]*model.MethodEntry, error) {
	iface := intf.intf.Type().Underlying().(*types.Interface)
	mset := types.NewMethodSet(iface)

	// Check for empty interfaces - they should be rejected
	if mset.Len() == 0 {
		return nil, fmt.Errorf("%v: interface %s has no methods - expected declaration",
			p.fset.Position(intf.intf.Pos()), intf.intf.Name())
	}

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
