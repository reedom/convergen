package model

type User struct {
	ID     int64
	Name   string
	Status string
	Origin Origin
}

type Origin struct {
	Region string
}
