package order_domain

import (
	"errors"
	"strings"

	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

// Item is owned by OrderStore; its state can only change through aggregate behavior.
type Item struct {
	entity ddd.Entity[ItemProps]
}

func NewItem(p ItemProps) (Item, error) {
	e, err := ddd.NewEntity(ddd.NewID(), p, ValidateItem)
	if err != nil {
		return Item{}, err
	}

	return Item{entity: *e}, nil
}

func ValidateItem(p ItemProps) error {
	if strings.TrimSpace(p.Title) == "" || strings.TrimSpace(p.Unit) == "" || p.Count <= 0 {
		return errors.New("item title, unit and positive count are required")
	}

	return nil
}

func (m Item) ID() ddd.ID {
	return m.entity.ID()
}

func (m Item) Props() ItemProps {
	p := m.entity.Props()

	return p
}

func (m Item) Validate() error {
	if m.ID().Empty() {
		return errors.New("item is required")
	}

	return ValidateItem(m.Props())
}
