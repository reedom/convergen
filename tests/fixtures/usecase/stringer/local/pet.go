package local

import (
	"strconv"
)

type Pet struct {
	ID     uint64
	Name   Name
	status Status
}

type Name string

func (n Name) String() string {
	return string(n)
}

func (p *Pet) Status() Status {
	return p.status
}

type Status int

func (s Status) String() string {
	return strconv.Itoa(int(s))
}
