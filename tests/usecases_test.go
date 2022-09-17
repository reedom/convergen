package testing

import (
	"fmt"
	"os"
	"testing"

	"github.com/reedom/convergen/pkg/generator"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUseCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source   string
		expected string
	}{
		//{
		//	source:   "fixtures/setups/getter/setup.go",
		//	expected: "fixtures/setups/getter/setup.gen.go",
		//},
		{
			source:   "fixtures/setups/style/setup.go",
			expected: "fixtures/setups/style/setup.gen.go",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.source, func(t *testing.T) {
			expected, err := os.ReadFile(tt.expected)
			require.Nil(t, err)

			logger.SetupLogger(logger.Enable())
			p, err := parser.NewParser(tt.source)
			require.Nil(t, err)
			code, err := p.Parse()
			require.Nil(t, err)

			g := generator.NewGenerator(code)
			actual, err := g.Generate(tt.source, false, true)
			require.Nil(t, err)

			assert.Equal(t, string(expected), string(actual))
			fmt.Println("-----------[generated]------------")
			fmt.Println(string(actual))
		})
	}
}
