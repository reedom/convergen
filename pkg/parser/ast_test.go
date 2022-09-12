package parser

import (
	"bytes"
	"go/printer"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetComments(t *testing.T) {
	t.Parallel()

	src := `
package main

// x is.
var x = 0

// Comment I-1
// loki:command i
// Comment I-2
type loki interface {
    // Comment M-1
	// loki;command m
	// Comment M-2
	ToModel()
}

// y is.
var y = 0
`
	p, err := NewParser("comment.go", src)
	require.Nil(t, err)
	intf, err := p.entryFile.getInterface()
	require.Nil(t, err)

	found := astGetDocCommentOn(p.entryFile.file, intf)
	require.NotNil(t, found)
	assert.Len(t, found.List, 3)
	assert.Equal(t, "// Comment I-1", found.List[0].Text)
}

func TestGetEmptyComments(t *testing.T) {
	t.Parallel()

	src := `
package main

// x is.
var x = 0

type loki interface {
    // Comment M-1
	ToModel()
}

// y is.
var y = 0
`
	p, err := NewParser("comment.go", src)
	require.Nil(t, err)
	intf, err := p.entryFile.getInterface()
	require.Nil(t, err)

	found := astGetDocCommentOn(p.entryFile.file, intf)
	assert.Nil(t, found)
}

func TestRemoveMatchComments(t *testing.T) {
	t.Parallel()

	src := `package main

// remain
// remove
// remain
var x = 1

// remove
// remain
var y = 1
`

	expected := `package main

// remain
// remain
var x = 1

// remain
var y = 1
`
	re := regexp.MustCompile(`//\s*remove\b`)

	p, err := NewParser("comment.go", src)
	require.Nil(t, err)
	e := p.entryFile
	astRemoveMatchComments(e.file, re)
	buf := bytes.Buffer{}
	require.Nil(t, printer.Fprint(&buf, e.fileSet, e.file))
	assert.Equal(t, expected, buf.String())
}
