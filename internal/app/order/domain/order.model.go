package order_domain

import (
	"errors"
	"strings"

	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

// Order is owned by OrderStore; its state can only change through aggregate behavior.
type Order struct {
	entity ddd.Entity[OrderProps]
}

func NewOrder(p OrderProps) (Order, error) {
	e, err := ddd.NewEntity(ddd.NewID(), p, ValidateOrder)
	if err != nil {
		return Order{}, err
	}

	return Order{entity: *e}, nil
}

func ValidateOrder(p OrderProps) error {
	if strings.TrimSpace(p.OrderId) == "" {
		return errors.New("order ID is required")
	}

	if !p.Customer.ID().Empty() {
		return p.Customer.Validate()
	}

	return nil
}

func (m Order) ID() ddd.ID {
	return m.entity.ID()
}

func (m Order) Props() OrderProps {
	p := m.entity.Props()

	return p
}

func (m Order) Validate() error {
	if m.ID().Empty() {
		return errors.New("order is required")
	}

	return ValidateOrder(m.Props())
}
