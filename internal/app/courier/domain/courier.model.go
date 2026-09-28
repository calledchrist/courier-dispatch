package courier_domain

import (
	"errors"
	"strings"
	"time"

	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

type Courier struct {
	aggregate ddd.Aggregate[CourierProps]
}

func NewCourier(p CreateCourierProps) (*Courier, error) {
	if p.ActiveTripLimit == 0 {
		p.ActiveTripLimit = DefaultActiveTripLimit
	}

	if p.Status == "" {
		p.Status = CourierOffline
	}

	props := CourierProps{
		Name:            p.Name,
		Mobile:          p.Mobile,
		Avatar:          p.Avatar,
		City:            p.City,
		HasPOSDevice:    p.HasPOSDevice,
		Status:          p.Status,
		VehicleType:     p.VehicleType,
		ActiveTripLimit: p.ActiveTripLimit,
	}
	a, err := ddd.NewAggregate(ddd.NewID(), props, ValidateCourier)
	if err != nil {
		return nil, err
	}

	return &Courier{aggregate: *a}, nil
}

func ValidateCourier(p CourierProps) error {
	if strings.TrimSpace(p.Name) == "" || strings.TrimSpace(p.City) == "" || len(strings.TrimSpace(p.Mobile)) < 10 {
		return errors.New("courier name, city and valid mobile are required")
	}

	if p.Status != CourierOnline && p.Status != CourierOffline && p.Status != CourierSuspended {
		return errors.New("invalid courier status")
	}

	if p.VehicleType != VehicleBike && p.VehicleType != VehicleCar {
		return errors.New("invalid courier vehicle")
	}

	if p.ActiveTripLimit <= 0 || len(p.ActiveDeliveryIDs) > p.ActiveTripLimit {
		return errors.New("invalid courier capacity")
	}

	seen := map[ddd.ID]bool{}
	for _, id := range p.ActiveDeliveryIDs {
		if id.Empty() || seen[id] {
			return errors.New("invalid active delivery references")
		}

		seen[id] = true
	}

	return nil
}

func (c *Courier) ID() ddd.ID {
	return c.aggregate.ID()
}

func (c *Courier) Version() uint64 {
	return c.aggregate.Version()
}

func (c *Courier) Props() CourierProps {
	p := c.aggregate.Props()
	p.ActiveDeliveryIDs = append([]ddd.ID(nil), p.ActiveDeliveryIDs...)

	return p
}

func (c *Courier) Clone() *Courier {
	copy := *c

	return &copy
}

func (c *Courier) ActiveTrips() int {
	return len(c.Props().ActiveDeliveryIDs)
}

func (c *Courier) Available() bool {
	p := c.Props()

	return p.Status == CourierOnline && p.Location.Validate() == nil && c.ActiveTrips() < p.ActiveTripLimit
}

func (c *Courier) UpdateLocation(location Location, at time.Time) error {
	if err := location.Validate(); err != nil {
		return err
	}

	if at.IsZero() || at.Before(c.Props().LocationUpdatedAt) {
		return errors.New("location timestamp must be present and chronological")
	}

	c.aggregate.Update(func(p *CourierProps) {
		p.Location = location
		p.LocationUpdatedAt = at
	})

	return nil
}

func (c *Courier) ChangeStatus(status CourierStatus) error {
	p := c.Props()
	p.Status = status
	if err := ValidateCourier(p); err != nil {
		return err
	}

	c.aggregate.Update(func(current *CourierProps) { *current = p })

	return nil
}

func (c *Courier) SetPOSDevice(has bool) {
	c.aggregate.Update(func(p *CourierProps) { p.HasPOSDevice = has })
}

func (c *Courier) SetActiveTripLimit(limit int) error {
	p := c.Props()
	p.ActiveTripLimit = limit
	if err := ValidateCourier(p); err != nil {
		return err
	}

	c.aggregate.Update(func(current *CourierProps) { *current = p })

	return nil
}

func (c *Courier) ReserveDelivery(id ddd.ID) error {
	if id.Empty() {
		return errors.New("delivery ID is required")
	}

	if !c.Available() {
		return errors.New("courier is unavailable or at capacity")
	}

	p := c.Props()
	for _, active := range p.ActiveDeliveryIDs {
		if active == id {
			return errors.New("delivery already reserved")
		}
	}

	p.ActiveDeliveryIDs = append(p.ActiveDeliveryIDs, id)
	c.aggregate.Update(func(current *CourierProps) { *current = p })

	return nil
}

func (c *Courier) ReleaseDelivery(id ddd.ID) error {
	p := c.Props()
	for i, active := range p.ActiveDeliveryIDs {
		if active == id {
			p.ActiveDeliveryIDs = append(p.ActiveDeliveryIDs[:i], p.ActiveDeliveryIDs[i+1:]...)
			c.aggregate.Update(func(current *CourierProps) { *current = p })

			return nil
		}
	}

	return errors.New("delivery is not reserved by courier")
}
