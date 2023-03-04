package util_test

import (
	"go/types"
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

func TestImportNames_TypeName(t *testing.T) {
	// Define the source code to test.
	source := `
		package main

		import (
			"fmt"
			"time"
		)

		type MyInt int

		var now = time.Now()

		func main() {
			fmt.Println(now)
			var x MyInt
			fmt.Println(x)
		}
	`
	file, _, pkg := loadSrc(t, source)

	// Create an ImportNames object.
	imports := util.NewImportNames(file.Imports)

	// Test TypeName with basic types.
	assert.Equal(t, "int", imports.TypeName(types.Typ[types.Int]))
	assert.Equal(t, "bool", imports.TypeName(types.Typ[types.Bool]))
	assert.Equal(t, "string", imports.TypeName(types.Typ[types.String]))
	assert.Equal(t, "float64", imports.TypeName(types.Typ[types.Float64]))
	assert.Equal(t, "complex128", imports.TypeName(types.Typ[types.Complex128]))

	// Test TypeName with named types in the same package.
	namedType := pkg.Scope().Lookup("MyInt").Type()
	assert.Equal(t, "MyInt", imports.TypeName(namedType))
	assert.False(t, imports.IsExternal(namedType))

	// Test TypeName with named types in external packages.
	namedType = pkg.Scope().Lookup("now").Type()
	assert.Equal(t, "time.Time", imports.TypeName(namedType))
	assert.True(t, imports.IsExternal(namedType))

	path, ok := imports.LookupName("time")
	assert.True(t, ok)
	assert.NotEmpty(t, path)
}
