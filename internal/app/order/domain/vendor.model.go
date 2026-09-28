package order_domain

import (
	"errors"
	"strings"

	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

// Vendor is owned by OrderStore; its state can only change through aggregate behavior.
type Vendor struct {
	entity ddd.Entity[VendorProps]
}

func NewVendor(p VendorProps) (Vendor, error) {
	p.Items = append([]Item(nil), p.Items...)
	e, err := ddd.NewEntity(ddd.NewID(), p, ValidateVendor)
	if err != nil {
		return Vendor{}, err
	}

	return Vendor{entity: *e}, nil
}

func ValidateVendor(p VendorProps) error {
	if p.VendorId == 0 || strings.TrimSpace(p.Title) == "" || strings.TrimSpace(p.AddressText) == "" || strings.TrimSpace(p.City) == "" {
		return errors.New("vendor ID, title, address and city are required")
	}

	if err := p.Location.Validate(); err != nil {
		return err
	}

	for _, item := range p.Items {
		if err := item.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (m Vendor) ID() ddd.ID {
	return m.entity.ID()
}

func (m Vendor) Props() VendorProps {
	p := m.entity.Props()
	p.Items = append([]Item(nil), p.Items...)

	return p
}

func (m Vendor) Validate() error {
	if m.ID().Empty() {
		return errors.New("vendor is required")
	}

	return ValidateVendor(m.Props())
}
