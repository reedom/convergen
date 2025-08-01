package domain

import (
	"errors"
	"fmt"
)

// Static errors for err113 compliance.
var (
	ErrInvalidPetStatusValue = errors.New("invalid value for PetStatus")
)

type PetStatus string

func NewPetStatusFromValue(value string) (PetStatus, error) {
	for _, s := range PetStatusValues {
		if s.String() == value {
			return s, nil
		}
	}

	return PetStatus(""), fmt.Errorf("%w: %v", ErrInvalidPetStatusValue, value)
}

func (s PetStatus) String() string {
	return string(s)
}

var (
	PetAvailable = PetStatus("available")
	PetPending   = PetStatus("pending")
	PetSold      = PetStatus("sold")
)

var PetStatusValues = []PetStatus{
	PetAvailable,
	PetPending,
	PetSold,
}
