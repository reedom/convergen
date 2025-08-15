package main

//go:generate go run github.com/reedom/convergen/v9

type User struct {
	Name string
}

type UserModel struct {
	Name string
}

type Convergen interface {
}
