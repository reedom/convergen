package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetComments(t *testing.T) {
	src := `
package main

// x is.
var x = 0

// Comment I-1
// convergen:command i
// Comment I-2
type Convergen interface {
    // Comment M-1
	// convergen;command m
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
	src := `
package main

// x is.
var x = 0

type Convergen interface {
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
