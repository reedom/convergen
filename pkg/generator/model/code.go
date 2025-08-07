// Package model provides the data structures used to represent the generated code.
package model

// Code represents the generated code.
type Code struct {
	// PackageName is the name of the package.
	BaseCode string
	// FunctionsBlock is the generated code for the functions.
	FunctionBlocks []FunctionsBlock
}
