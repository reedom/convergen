package util_test

import (
	"testing"

	"github.com/reedom/convergen/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestImportNames(t *testing.T) {
	t.Parallel()

	src := `
package main

import (
	js "encoding/json"
	"fmt"
	"net/http"
)

func main() {
	v, _ := js.Marshal([]string{"Hello", "world"})
	fmt.Println(v)
	fmt.Println(http.StatusOK)
}
`
	file, _, _ := loadSrc(t, src)
	names := util.NewImportNames(file.Imports)
	for path, expected := range map[string]string{
		"fmt":           "fmt",
		"encoding/json": "js",
		"net/http":      "http",
	} {
		actual, ok := names.LookupName(path)
		if assert.True(t, ok) {
			assert.Equal(t, expected, actual)
		}
	}

	_, ok := names.LookupName("unknown")
	assert.False(t, ok)
}
