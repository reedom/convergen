package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerator(t *testing.T) {
	c, err := NewParser("../../tests/fixtures/usecase/types/setup.go")
	require.Nil(t, err)
	intf, err := c.extractIntfEntry()
	require.Nil(t, err)
	_, err = c.parseMethods(intf)
	require.Nil(t, err)
}
