//go:build convergen
// +build convergen

package mapname

import (
	"github.com/reedom/convergen/tests/fixtures/data/domain"
	"github.com/reedom/convergen/tests/fixtures/data/model"
)

//go:generate go run github.com/reedom/convergen
type Convergen interface {
	// :map ID CategoryID
	// :typecast
	DomainToModel(cat *domain.Category) *model.Category
}
