package model

// Var represents a defined variable.
type Var struct {
	// Name is the name of the variable.
	Name string
	// Type represents the type expression of the variable without pointer mark "*".
	// If the type is of an external package, Type has its package name in the code, too. (e.g. model.User)
	Type string
	// Pointer indicates whether the variable is defined as a pointer.
	Pointer bool
	// External indicates whether the Type is defined in an external package.
	External bool
}

// FullType creates a complete type expression string that can be used for var declaration.
// E.g. "*model.Pet" for "var x *model.Pet".
func (v Var) FullType() string {
	if v.Pointer {
		return "*" + v.Type
	}
	return v.Type
}

// PtrLessFullType creates a complete type expression string but omits the pointer "*" symbol at the top.
func (v Var) PtrLessFullType() string {
	return v.Type
}
