//go:build convergen

package simple

import (
	"github.com/reedom/convergen/pkg/tests/fixtures/data/domain"
	"github.com/reedom/convergen/pkg/tests/fixtures/data/model"
)

//go:generate go run github.com/reedom/convergen
type convergen interface {
	DomainToModel(pet *domain.Pet) *model.Pet
}
