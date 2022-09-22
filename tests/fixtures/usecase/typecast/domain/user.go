package domain

import (
	"github.com/reedom/convergen/tests/fixtures/usecase/typecast/enums"
)

type User struct {
	ID     int
	Name   string
	Status enums.Status
}
