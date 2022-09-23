package parser

import (
	"go/types"
	"unicode"

	gonanoid "github.com/matoous/go-nanoid"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/option"
	"github.com/reedom/convergen/pkg/util"
)

const intfName = "Convergen"

type intfEntry struct {
	intf   types.Object
	opts   option.Options
	marker string
}

// findConvergenEntries collects convergen interfaces from the setup file.
// The target interface form either in the name of "Convergen" or having ":convergen" notation in its Doc comments.
// For them, this function also parses notations in their doc comments.
func (p *Parser) findConvergenEntries() ([]*intfEntry, error) {
	entries := make([]*intfEntry, 0)
	scope := p.pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		_, ok := obj.Type().Underlying().(*types.Interface)
		if !ok {
			continue
		}
		if p.srcPath != p.fset.Position(obj.Pos()).Filename {
			// Skip other than the entry file.
			continue
		}

		docComment, cleanUp := util.GetDocCommentOn(p.file, obj)

		isTarget := obj.Name() == intfName || util.MatchComments(docComment, reConvergen)
		if !isTarget {
			continue
		}

		logger.Printf("%v: target interface found: %v", p.fset.Position(obj.Pos()), obj.Name())

		notations := util.ExtractMatchComments(docComment, reNotation)
		if docComment != nil {
			docComment.List = nil
		}
		cleanUp()

		opts := p.opts
		err := p.parseNotationInComments(notations, option.ValidOpsIntf, &opts)
		if err != nil {
			return nil, err
		}

		marker, _ := gonanoid.Nanoid()
		entry := &intfEntry{
			intf:   obj,
			opts:   p.opts,
			marker: marker,
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, logger.Errorf("%v: %v interface not found", p.fset.Position(p.file.Package), intfName)
	}

	return entries, nil
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
