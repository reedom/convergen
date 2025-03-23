package model

// FunctionsBlock represents a group of functions.
type FunctionsBlock struct {
	Marker    string      // Marker is a special comment marker for indicating a specific section of functions.
	Functions []*Function // Functions is the list of functions.
}

// Function represents a function.
type Function struct {
	Comments       []string     // Comments is the list of comment lines before the function definition.
	Name           string       // Name is the function name.
	Receiver       string       // Receiver is the receiver type name, if any.
	Src            Var          // Src is the source variable.
	Dst            Var          // Dst is the destination variable.
	AdditionalArgs []Var        // AdditionalArgs is the additional arguments variables.
	RetError       bool         // RetError indicates whether the function returns an error.
	DstVarStyle    DstVarStyle  // DstVarStyle is the style of the destination variable declaration.
	Assignments    []Assignment // Assignments is the list of assignments in the function body.
	PreProcess     *Manipulator // PreProcess is the function that is applied before the assignments.
	PostProcess    *Manipulator // PostProcess is the function that is applied after the assignments.
}
