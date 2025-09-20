package domain

import (
	"github.com/reedom/convergen/v8/tests/fixtures/usecase/typecast/enums"
)

type User struct {
	ID     int
	Name   string
	Status enums.Status
	Origin Origin
}

type Origin struct {
	Region string
}
