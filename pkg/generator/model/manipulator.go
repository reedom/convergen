package model

import "fmt"

// Manipulator represents a function that manipulates a value.
type Manipulator struct {
	Pkg               string // Pkg is the package name of the function.
	Name              string // Name is the name of the function.
	IsDstPtr          bool   // IsDstPtr indicates that the first argument is a pointer to the destination.
	IsSrcPtr          bool   // IsSrcPtr indicates that the second argument is a pointer to the source.
	HasAdditionalArgs bool   // HasAdditionalArgs indicates whether the function has additional arguments.
	RetError          bool   // RetError indicates that the function returns an error.
}

// FuncName returns the fully qualified name of the function.
func (m *Manipulator) FuncName() string {
	if m.Pkg != "" {
		return fmt.Sprintf("%v.%v", m.Pkg, m.Name)
	}
	return m.Name
}
