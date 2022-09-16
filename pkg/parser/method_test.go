package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerator(t *testing.T) {
	c, err := NewParser("../fixtures/setups/types/setup.go")
	require.Nil(t, err)
	err = c.extractIntfEntry()
	require.Nil(t, err)
	_, err = c.parseMethods()
	require.Nil(t, err)
}
