package ddd

import (
	"errors"
	"fmt"
)

type Entity[T any] struct {
	id    ID
	props T
}

func NewEntity[T any](
	id ID,
	props T,
	validate func(T) error,
) (*Entity[T], error) {
	if id.Empty() {
		return &Entity[T]{}, errors.New("entity ID must not be empty")
	}

	if !IsObject(props) {
		return nil, errors.New("model props should be an object")
	}

	if HasTooFewProps(props, MIN_PROPS) {
		return nil, fmt.Errorf(
			"model props must have at least %d properties",
			MIN_PROPS,
		)
	}

	if HasTooManyProps(props, MAX_PROPS) {
		return nil, fmt.Errorf(
			"model props must not have more than %d properties",
			MAX_PROPS,
		)
	}

	if validate != nil {
		if err := validate(props); err != nil {
			return &Entity[T]{}, err
		}
	}

	return &Entity[T]{
		id: id,

		props: props,
	}, nil
}

func (e *Entity[T]) ID() ID {
	return e.id
}

func (e *Entity[T]) Props() T {
	return e.props
}

func (e *Entity[T]) Update(fn func(*T)) {
	fn(&e.props)
}
