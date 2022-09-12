package parser_test

import (
	"testing"

	"github.com/reedom/loki/pkg/parser"
	"github.com/stretchr/testify/require"
)

func TestParser_Parse(t *testing.T) {
	p, err := parser.NewParser("../fixtures/setups/getter/setup.go", nil)
	require.Nil(t, err)
	require.Nil(t, p.Parse())
}
