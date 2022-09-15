package convergen_test

import (
	"testing"

	"github.com/reedom/convergen/pkg/convergen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoConvergenInterface(t *testing.T) {
	c, err := convergen.NewConvergen("../fixtures/setups/nointf/setup.go")
	require.Nil(t, err)
	err = c.ExtractIntfEntry()
	require.NotNil(t, err)
	assert.ErrorContains(t, err, "nointf/setup.go:1:1: Convergen interface not found")
}
