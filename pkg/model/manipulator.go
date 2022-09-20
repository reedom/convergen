package model

import (
	"fmt"
)

type Manipulator struct {
	Pkg          string
	Name         string
	IsDstPtr     bool
	IsSrcPtr     bool
	ReturnsError bool
}

func (m *Manipulator) FuncName() string {
	if m.Pkg != "" {
		return fmt.Sprintf("%v.%v", m.Pkg, m.Name)
	}
	return m.Name
}
