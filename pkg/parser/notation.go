package parser

import (
	"regexp"
	"strings"
)

type notationKind string

const (
	ntInvalid     = notationKind("")
	ntOpt         = notationKind("opt")
	ntRcv         = notationKind("rcv")
	ntSkip        = notationKind("skip")
	ntMap         = notationKind("map")
	ntTag         = notationKind("tag")
	ntConv        = notationKind("conv")
	ntPostProcess = notationKind("postProcess")
)

var validNotation = map[string]struct{}{
	string(ntOpt):         {},
	string(ntRcv):         {},
	string(ntSkip):        {},
	string(ntMap):         {},
	string(ntTag):         {},
	string(ntConv):        {},
	string(ntPostProcess): {},
}

type notation interface {
	kind() notationKind
}

func isInvalidNotation(n notation) bool {
	return n.kind() == ntInvalid
}

type invalidNotation struct {
	notation
}

func (n invalidNotation) kind() notationKind {
	return ntInvalid
}

type optNotation struct {
	notation
}

func (o optNotation) kind() notationKind {
	return ntOpt
}

var reNotation = regexp.MustCompile(`^\s*//\s*loki:(\S*)\s*(.*)$`)

func newNotion(comment string) notation {
	m := reNotation.FindStringSubmatch(comment)
	if m == nil {
		return nil
	}

	if len(m[1]) == 0 {
		return invalidNotation{}
	}

	fields := strings.Fields(m[1])
}
