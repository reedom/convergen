package convergen_test

import (
	"testing"

	"github.com/reedom/convergen/pkg/convergen"
	"github.com/stretchr/testify/require"
)

func TestGenerator(t *testing.T) {
	c, err := convergen.NewConvergen("../fixtures/setups/types/setup.go")
	require.Nil(t, err)
	err = c.ExtractIntfEntry()
	require.Nil(t, err)
	err = c.Generate("testit.go")
	require.Nil(t, err)
}
