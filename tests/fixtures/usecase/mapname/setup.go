//go:build convergen
// +build convergen

package mapname

import (
	"github.com/reedom/convergen/v8/tests/fixtures/data/domain"
	"github.com/reedom/convergen/v8/tests/fixtures/data/model"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :map Category.ID Category.CategoryID
	// :map Status.String() Status
	// :typecast
	DomainToModel(*domain.Pet) *model.Pet
}
