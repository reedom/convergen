package testing

import (
	"fmt"
	"os"
	"testing"

	"github.com/reedom/convergen/pkg/generator"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/model"
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
		{
			source:   "fixtures/usecase/converter/setup.go",
			expected: "fixtures/usecase/converter/setup.gen.go",
		},
		{
			source:   "fixtures/usecase/getter/setup.go",
			expected: "fixtures/usecase/getter/setup.gen.go",
		},
		{
			source:   "fixtures/usecase/nocase/setup.go",
			expected: "fixtures/usecase/nocase/setup.gen.go",
		},
		{
			source:   "fixtures/usecase/mapname/setup.go",
			expected: "fixtures/usecase/mapname/setup.gen.go",
		},
		{
			source:   "fixtures/usecase/postprocess/setup.go",
			expected: "fixtures/usecase/postprocess/setup.gen.go",
		},
		{
			source:   "fixtures/usecase/simple/setup.go",
			expected: "fixtures/usecase/simple/setup.gen.go",
		},
		{
			source:   "fixtures/usecase/stringer/setup.go",
			expected: "fixtures/usecase/stringer/setup.gen.go",
		},
		{
			source:   "fixtures/usecase/style/setup.go",
			expected: "fixtures/usecase/style/setup.gen.go",
		},
		{
			source:   "fixtures/usecase/typecast/setup.go",
			expected: "fixtures/usecase/typecast/setup.gen.go",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.source, func(t *testing.T) {
			t.Parallel()
			expected, err := os.ReadFile(tt.expected)
			require.Nil(t, err)

			if tt.source == "fixtures/usecase/xxxx/setup.go" {
				logger.SetupLogger(logger.Enable())
			}

			p, err := parser.NewParser(tt.source)
			require.Nil(t, err)
			methods, err := p.Parse()
			require.Nil(t, err)
			builder := p.CreateBuilder()
			functions, err := builder.CreateFunctions(methods)
			require.Nil(t, err)

			pre, post, err := p.GenerateBaseCode()
			require.Nil(t, err)

			code := model.Code{
				Pre:       pre,
				Post:      post,
				Functions: functions,
			}
			g := generator.NewGenerator(code)
			actual, err := g.Generate(tt.source, false, true)
			require.Nil(t, err)

			if !assert.Equal(t, string(expected), string(actual)) {
				fmt.Println("-----------[generated]------------")
				fmt.Println(string(actual))
			}
		})
	}
}
