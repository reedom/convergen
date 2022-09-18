package parser

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/reedom/convergen/pkg/generator"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestParser_Parse(t *testing.T) {
	logger.SetupLogger(logger.Enable())
	p, err := NewParser("../../tests/fixtures/usecase/getter/setup.go")
	require.Nil(t, err)
	code, err := p.Parse()
	require.Nil(t, err)

	g := generator.NewGenerator(code)
	wd, _ := os.Getwd()
	absPath := path.Join(wd, "../fixtures/usecase/getter/generated.go")
	generated, err := g.Generate(absPath, false, true)
	require.Nil(t, err)
	fmt.Println(string(generated))
}
