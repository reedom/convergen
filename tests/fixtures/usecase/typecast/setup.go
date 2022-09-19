//go:build convergen
// +build convergen

package stringer

import (
	"github.com/reedom/convergen/tests/fixtures/data/domain"
	"github.com/reedom/convergen/tests/fixtures/data/model"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :typecast
	DomainToModel(*domain.Pet) *model.Pet
	// :typecast
	ModelToDomain(*model.Pet) *domain.Pet
}
