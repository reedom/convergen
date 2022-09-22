package option

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
				{src: "Field", dst: "domain.Field", exactCase: true, matches: false},
			},
			"Label": {
				{src: "Field", dst: "Label", exactCase: true, matches: true},
				{src: "Field", dst: "domain.Label", exactCase: true, matches: false},
			},
		},
	}

	for srcPtn, ttMap := range cases {
		for dstPtn, ttList := range ttMap {
			for _, tt := range ttList {
				m := NewNameMatcher(srcPtn, dstPtn, 0)
				assert.Equal(t, tt.matches, m.Match(tt.src, tt.dst, tt.exactCase),
					`pattern src="%v", dst="%v" against src="%v", dst="%v" (case-sensitive=%v)`,
					srcPtn, dstPtn, tt.src, tt.dst, tt.exactCase)
			}
		}
	}
}
