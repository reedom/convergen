package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoConvergenInterface(t *testing.T) {
	// Create a temporary test file with no convergen interface
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "setup.go")

	content := `package test

type RegularStruct struct {
	Name string
}

type RegularInterface interface {
	DoSomething() string
}
`

	err := os.WriteFile(testFile, []byte(content), 0o600)
	require.NoError(t, err)

	c, err := NewParser(testFile, testFile+".gen.go")
	require.Nil(t, err)

	_, err = c.findConvergenEntries()
	require.NotNil(t, err)
	assert.ErrorContains(t, err, "Convergen interface not found")
}
