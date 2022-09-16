package parser

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/reedom/convergen/pkg/generator"
	"github.com/stretchr/testify/require"
)

func TestParser_Parse(t *testing.T) {
	p, err := NewParser("../fixtures/setups/getter/setup.go")
	require.Nil(t, err)
	code, err := p.Parse()
	require.Nil(t, err)

	g := generator.NewGenerator(code)
	wd, _ := os.Getwd()
	absPath := path.Join(wd, "../fixtures/setups/getter/generated.go")
	generated, err := g.Generate(absPath, true)
	require.Nil(t, err)
	fmt.Println(string(generated))
}
