package test

//go:generate go run github.com/reedom/convergen/v9

type User struct {
	Name string
}

type UserModel struct {
	Name string
}

type CircularConstraint[T any] interface {
	Method() T
}

type Convergen[T CircularConstraint[T], U CircularConstraint[U]] interface {
	// Circular constraints - should handle gracefully
	Convert(*User) *UserModel
}