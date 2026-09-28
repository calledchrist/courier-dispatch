package order_domain

import "github.com/calledchrist/courier-dispatch/internal/shared/geo"

type Location = geo.Location

func NewLocation(p LocationProps) (Location, error) {
	return geo.NewLocation(p)
}

func ValidateLocation(p LocationProps) error {
	return geo.ValidateLocation(p)
}
