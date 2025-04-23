//go:build convergen
// +build convergen

package converter

import (
	"github.com/reedom/convergen/tests/fixtures/data/model"
	"github.com/reedom/convergen/tests/fixtures/data/model/abc222"
)

//go:generate go run github.com/reedom/convergen


type Convergen interface {
	//:map $2 List
	DomainToModel(*model.Additional,[]abc222.Additional321) *abc222.AdditionalItem123
}
