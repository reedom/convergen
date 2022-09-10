package domain

import (
	"fmt"
)

type Category struct {
	id   uint
	name string
}

func NewCategory(id uint, name string) (*Category, error) {
	if name == "" {
		return nil, fmt.Errorf("category name is empty")
	}
	return &Category{id: id, name: name}, nil
}

func (c Category) ID() uint {
	return c.id
}

func (c Category) Name() string {
	return c.name
}
