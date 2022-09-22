package option

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIdentMatcher(t *testing.T) {
	t.Parallel()

	cases := map[string][]struct {
		ident     string
		exactCase bool
		matches   bool
	}{
		"Field": {
			{ident: "Field", exactCase: true, matches: true},
			{ident: "field", exactCase: true, matches: false},
			{ident: "field", exactCase: false, matches: true},
			{ident: "aField", exactCase: true, matches: false},
			{ident: "Fields", exactCase: true, matches: false},
		},
		"model.Pet.Name": {
			{ident: "model.Pet.Name", exactCase: true, matches: true},
			{ident: "model.Pet.name", exactCase: false, matches: true},
			{ident: "model_Pet_Name", exactCase: false, matches: false},
		},
	}

	for pattern, ttList := range cases {
		for _, tt := range ttList {
			m := NewIdentMatcher(pattern)
			assert.Equal(t, tt.matches, m.Match(tt.ident, tt.exactCase),
				`pattern "%v" against "%v" (case-sensitive=%v)`, pattern, tt.ident, tt.exactCase)
		}
	}
}
