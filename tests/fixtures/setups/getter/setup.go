//go:build convergen
// +build convergen

package simple

import (
	"github.com/reedom/convergen/tests/fixtures/data/ddd/domain"
	"github.com/reedom/convergen/tests/fixtures/data/ddd/model"
)

// convergen:opt:style return
//
//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// convergen:map foo bar
	// DomainToModel copies domain.Pet to model.Pet.
	DomainToModel(pet *domain.Pet) *model.Pet
}
