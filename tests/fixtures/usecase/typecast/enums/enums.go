package enums

type Status string

const (
	NotVerified = Status("notVerified")
	Verified    = Status("verified")
	Invalidated = Status("invalidated")
)

func (s Status) String() string {
	return string(s)
}

type Class struct {
	class string
}

func (c Class) String() string {
	return c.class
}

var (
	Basic   = Class{class: "basic"}
	Premier = Class{class: "premier"}
)
