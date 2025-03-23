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

type A struct {
	JSONDate JSONDate
	Value    int
}

type B struct {
	JSONDate time.Time
	Value    int
	Arg0     uint
	Arg1     string
}

type AdditionalArgs struct {
	Arg1 string
}

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :map JSONDate.Time() JSONDate
	FromTo(*From) *To
	// :map JSONDate.Time() JSONDate
	// :map $1.Value Value
	// :map $2 Arg0
	// :map $3.Arg1  Arg1
	AToB(*A, uint, AdditionalArgs) *B

	// :style arg
	// :map JSONDate.Time() JSONDate
	// :map $1.Value Value
	// :map $2 Arg0
	// :map $3.Arg1  Arg1
	AToBArgStyle(*A, uint, AdditionalArgs) *B
}
