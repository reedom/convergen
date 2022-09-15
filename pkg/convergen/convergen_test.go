package convergen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_Parse(t *testing.T) {
	p, err := NewConvergen("../fixtures/setups/getter/setup.go")
	require.Nil(t, err)
	require.Nil(t, p.ExtractIntfEntry())
}
