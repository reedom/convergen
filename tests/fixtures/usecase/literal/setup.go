//go:build convergen

package literal

import (
	"github.com/reedom/convergen/v8/tests/fixtures/data/domain"
	"github.com/reedom/convergen/v8/tests/fixtures/data/model"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :literal  Name   "abc  def"
	DomainToModel(*domain.Pet) *model.Pet
	ModelToDomain(*model.Pet) *domain.Pet
}
