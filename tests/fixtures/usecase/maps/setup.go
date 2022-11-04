//go:build convergen

package maps

import (
	"time"
)

type From struct {
	JSONDate JSONDate
}

type To struct {
	JSONDate time.Time
}

type JSONDate time.Time

func (t *JSONDate) Time() time.Time {
	return time.Time(*t)
}

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :map JSONDate.Time() JSONDate
	FromTo(*From) *To
}
