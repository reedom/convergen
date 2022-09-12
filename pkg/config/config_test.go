package config_test

import (
	"os"
	"testing"

	"github.com/reedom/loki/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputPath(t *testing.T) {
	wd, err := os.Getwd()
	require.Nil(t, err)

	for output, expected := range map[string]string{
		"./generated/converter.go": wd + "/generated/converter.go",
		"":                         wd + "/setup.gen.go1",
	} {
		c := config.Config{Output: output}
		actual := c.OutputPath("setup.go")
		assert.Equal(t, expected, actual, `"Output": "%v"`, output)
	}
}
