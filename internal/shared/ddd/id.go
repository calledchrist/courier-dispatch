package ddd

import (
	"github.com/oklog/ulid/v2"
)

type ID string

func NewID() ID {
	return ID(ulid.Make().String())
}

func (id ID) String() string {
	return string(id)
}

func (id ID) Empty() bool {
	return id == ""
}
