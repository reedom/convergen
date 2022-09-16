package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_Parse(t *testing.T) {
	p, err := NewParser("../fixtures/setups/getter/setup.go")
	require.Nil(t, err)
	require.Nil(t, p.extractIntfEntry())
}
