// Package geo contains immutable geographic values shared by the bounded contexts.
package geo

import (
	"errors"
	"math"
)

type LocationProps struct {
	Latitude  float64
	Longitude float64
}

type Location struct {
	value LocationProps
	valid bool
}

func NewLocation(p LocationProps) (Location, error) {
	if err := ValidateLocation(p); err != nil {
		return Location{}, err
	}

	return Location{
		value: p,
		valid: true,
	}, nil
}

func ValidateLocation(p LocationProps) error {
	if math.IsNaN(p.Latitude) || math.IsInf(p.Latitude, 0) || p.Latitude < -90 || p.Latitude > 90 {
		return errors.New("invalid coordinates")
	}

	if math.IsNaN(p.Longitude) || math.IsInf(p.Longitude, 0) || p.Longitude < -180 || p.Longitude > 180 {
		return errors.New("invalid coordinates")
	}

	return nil
}

func (l Location) Validate() error {
	if !l.valid {
		return errors.New("location is required")
	}

	return ValidateLocation(l.value)
}

func (l Location) Value() LocationProps {
	return l.value
}

func (l Location) Equal(other Location) bool {
	return l == other
}

func (l Location) DistanceKM(other Location) float64 {
	a, b := l.value, other.value
	rad := math.Pi / 180
	dlat, dlon := (b.Latitude-a.Latitude)*rad, (b.Longitude-a.Longitude)*rad
	h := math.Pow(math.Sin(dlat/2), 2) + math.Cos(a.Latitude*rad)*math.Cos(b.Latitude*rad)*math.Pow(math.Sin(dlon/2), 2)

	return 6371 * 2 * math.Asin(math.Sqrt(math.Min(1, h)))
}
