package domain

import (
	"errors"
)

// Static errors for err113 compliance.
var (
	ErrCategoryNameEmpty = errors.New("category name is empty")
)

type Category struct {
	id   uint
	name string
}

func NewCategory(id uint, name string) (*Category, error) {
	if name == "" {
		return nil, ErrCategoryNameEmpty
	}

	return &Category{id: id, name: name}, nil
}

func (c Category) ID() uint {
	return c.id
}

func (c Category) Name() string {
	return c.name
}
