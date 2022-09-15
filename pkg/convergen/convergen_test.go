package convergen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_Parse(t *testing.T) {
	p, err := NewConvergen("../fixtures/setups/getter/setup.go")
	require.Nil(t, err)
	require.Nil(t, p.Parse())
}

func TestPathMatch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pattern   string
		path      string
		exactCase bool
		match     bool
	}{
		{"Name", "Name", true, true},
		{"Name", "Nam", true, false},
		{"Name", "name", true, false},
		{"name", "Name", true, false},
		{"name", "Name", false, true},
		{"Name", "name", false, true},
		{"Na*", "name", false, true},
		{"N*e", "name", false, true},
	}

	for i, tt := range cases {
		actual, err := pathMatch(tt.pattern, tt.path, tt.exactCase)
		require.Nil(t, err, `case %v has invalid pattern "%v"`, i, tt.pattern)
		if tt.match {
			assert.True(t, actual, `pattern "%v" against "%v" should match`)
		} else {
			assert.False(t, actual, `pattern "%v" against "%v" should not match`)
		}
	}
}
