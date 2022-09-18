package option

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNameMatcher(t *testing.T) {
	t.Parallel()

	cases := map[string]map[string][]struct {
		src       string
		dst       string
		exactCase bool
		matches   bool
	}{
		"Field": {
			"Field": {
				{src: "Field", dst: "Field", exactCase: true, matches: true},
				{src: "Field", dst: "field", exactCase: true, matches: false},
			},
			"": {
				{src: "Field", dst: "Field", exactCase: true, matches: true},
				{src: "Field", dst: "domain.Field", exactCase: true, matches: true},
				{src: "Field", dst: "domain.Field.X", exactCase: true, matches: false},
			},
			"Label": {
				{src: "Field", dst: "Label", exactCase: true, matches: true},
				{src: "Field", dst: "domain.Label", exactCase: true, matches: false},
			},
			`/.*\.Label$/`: {
				{src: "Field", dst: "domain.Label", exactCase: true, matches: true},
				{src: "Field", dst: "domain.Labels", exactCase: true, matches: false},
				{src: "Field", dst: "domain.label", exactCase: true, matches: false},
				{src: "Field", dst: "domain.label", exactCase: false, matches: true},
			},
		},
		"/^model.Field$/": {
			"": {
				{src: "model.Field", dst: "Field", exactCase: true, matches: true},
				{src: "model.Field", dst: "domain.Field", exactCase: true, matches: true},
				{src: "amodel.Field", dst: "domain.Field", exactCase: true, matches: false},
				{src: "model.Fields", dst: "domain.Field", exactCase: true, matches: false},
			},
		},
	}

	for srcPtn, ttMap := range cases {
		for dstPtn, ttList := range ttMap {
			for _, tt := range ttList {
				m, err := NewNameMatcher(srcPtn, dstPtn, tt.exactCase)
				require.Nil(t, err, `test pattern src="%v", dst="%v"`, srcPtn, dstPtn)
				assert.Equal(t, tt.matches, m.Match(tt.src, tt.dst, tt.exactCase),
					`pattern src="%v", dst="%v" against src="%v", dst="%v" (case-sensitive=%v)`,
					srcPtn, dstPtn, tt.src, tt.dst, tt.exactCase)
			}
		}
	}
}
