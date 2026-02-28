package idutil

import "github.com/google/uuid"

type ID[T any] struct{ uuid.UUID }

func NewID[T any]() (ID[T], error) {
	u, err := uuid.NewV7()
	if err != nil {
		return ID[T]{}, err
	}
	return ID[T]{UUID: u}, nil
}

func ParseID[T any](s string) (ID[T], error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return ID[T]{}, err
	}
	return ID[T]{UUID: u}, nil
}

func (id ID[T]) IsZero() bool   { return id.UUID == uuid.Nil }
func (id ID[T]) String() string { return id.UUID.String() }
