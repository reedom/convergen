package convergen

import (
	"regexp"
)

var reGoBuildGen = regexp.MustCompile(`\s*//\s*((go:generate\b|build convergen\b)|\+build convergen)`)
