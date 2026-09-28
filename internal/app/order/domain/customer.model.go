package order_domain

import (
	"errors"
	"strings"

	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

// Customer is owned by OrderStore; its state can only change through aggregate behavior.
type Customer struct {
	entity ddd.Entity[CustomerProps]
}

func NewCustomer(p CustomerProps) (Customer, error) {
	e, err := ddd.NewEntity(ddd.NewID(), p, ValidateCustomer)
	if err != nil {
		return Customer{}, err
	}

	return Customer{entity: *e}, nil
}

func ValidateCustomer(p CustomerProps) error {
	if strings.TrimSpace(p.Name) == "" || len(strings.TrimSpace(p.Mobile)) < 10 || strings.TrimSpace(p.AddressText) == "" {
		return errors.New("customer name, mobile and address are required")
	}

	return p.Location.Validate()
}

func (m Customer) ID() ddd.ID {
	return m.entity.ID()
}

func (m Customer) Props() CustomerProps {
	p := m.entity.Props()

	return p
}

func (m Customer) Validate() error {
	if m.ID().Empty() {
		return errors.New("customer is required")
	}

	return ValidateCustomer(m.Props())
}
