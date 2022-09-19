package parser

import (
	"go/types"
	"unicode"

	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
)

const intfName = "Convergen"

type intfEntry struct {
	intf *types.TypeName
	opts option.Options
}

// extractIntfEntry looks up the setup interface with the name of intfName("Convergen") and also
// parses convergen notations from the interface's doc comment.
func (p *Parser) extractIntfEntry() (*intfEntry, error) {
	intf, err := p.findIntfEntry(p.pkg.Types.Scope(), intfName)
	if err != nil {
		return nil, err
	}

	docComment, cleanUp := util.GetDocCommentOn(p.file, intf)
	notations := util.ExtractMatchComments(docComment, reNotation)
	docComment.List = nil
	cleanUp()

	opts := p.opts
	err = p.parseNotationInComments(notations, option.ValidOpsIntf, &opts)
	if err != nil {
		return nil, err
	}

	return &intfEntry{
		intf: intf,
		opts: p.opts,
	}, nil
}

// findIntfEntry looks up the setup interface with the specific name and returns it.
func (p *Parser) findIntfEntry(scope *types.Scope, name string) (*types.TypeName, error) {
	if typ := scope.Lookup(name); typ != nil {
		if intf, ok := typ.(*types.TypeName); ok {
			if _, ok = intf.Type().Underlying().(*types.Interface); ok {
				logger.Printf("%v: %v interface found", p.fset.Position(intf.Pos()), name)
				return intf, nil
			}
			return nil, logger.Errorf("%v: %v it not interface", p.fset.Position(intf.Pos()), name)
		}
	}
	return nil, logger.Errorf("%v: %v interface not found", p.fset.Position(p.file.Package), name)
}

func isValidIdentifier(id string) bool {
	for i, r := range id {
		if !unicode.IsLetter(r) &&
			!(i > 0 && unicode.IsDigit(r)) {
			return false
		}
	}
	return id != ""
}
