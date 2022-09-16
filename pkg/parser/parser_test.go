package parser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_Parse(t *testing.T) {
	p, err := NewParser("../fixtures/setups/getter/setup.go")
	require.Nil(t, err)
	f, err := p.Parse()
	require.Nil(t, err)
	fmt.Println(f.Pre)
	fmt.Println("------------")
	fmt.Println(f.Post)
}
