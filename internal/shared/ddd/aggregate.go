package ddd

import (
	"errors"
	"fmt"
)

type Aggregate[T any] struct {
	id      ID
	version uint64
	props   T
}

func NewAggregate[T any](
	id ID,
	props T,
	validate func(T) error,
) (*Aggregate[T], error) {
	if id.Empty() {
		return nil, errors.New("aggregate ID must not be empty")
	}

	if validate != nil {
		if err := validate(props); err != nil {
			return nil, err
		}
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

	return &Aggregate[T]{
		id: id,

		props: props,
	}, nil
}

func (a *Aggregate[T]) ID() ID {
	return a.id
}

func (a *Aggregate[T]) Version() uint64 {
	return a.version
}

func (a *Aggregate[T]) Props() T {
	return a.props
}

func (a *Aggregate[T]) Update(fn func(*T)) {
	fn(&a.props)
	a.version++
}
