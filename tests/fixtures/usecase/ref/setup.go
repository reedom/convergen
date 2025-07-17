//go:build convergen
// +build convergen

package ref

import (
	"github.com/reedom/convergen/v8/tests/fixtures/data/domain"
	"github.com/reedom/convergen/v8/tests/fixtures/data/model"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :conv CatDomainToModel Category
	DomainToModel(*domain.Pet) *model.Pet

	// :map ID CategoryID
	// :typecast
	CatDomainToModel(*domain.Category) model.Category
}
