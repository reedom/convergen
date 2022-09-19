//go:build convergen

package simple

import (
	"github.com/reedom/convergen/tests/fixtures/data/domain"
	"github.com/reedom/convergen/tests/fixtures/data/model"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	DomainToModel(*domain.Pet) *model.Pet
	ModelToDomain(*model.Pet) *domain.Pet
}
