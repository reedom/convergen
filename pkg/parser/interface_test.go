package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoConvergenInterface(t *testing.T) {
	c, err := NewParser("../fixtures/setups/nointf/setup.go")
	require.Nil(t, err)
	_, err = c.extractIntfEntry()
	require.NotNil(t, err)
	assert.ErrorContains(t, err, "nointf/setup.go:1:1: Convergen interface not found")
}
