package delivery_domain

import (
	"errors"
	"strings"
	"time"

	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
	"github.com/calledchrist/courier-dispatch/internal/shared/geo"
)

type Status string

const (
	Assigned  Status = "assigned"
	PickedUp  Status = "picked_up"
	Delivered Status = "delivered"
	Cancelled Status = "cancelled"
)

// Stop is an immutable address/location snapshot, independent of the order model.
type StopProps struct {
	Address  string
	Location geo.Location
}

type Stop struct {
	props StopProps
}

func NewStop(p StopProps) (Stop, error) {
	if err := ValidateStop(p); err != nil {
		return Stop{}, err
	}

	return Stop{props: p}, nil
}

func ValidateStop(p StopProps) error {
	if strings.TrimSpace(p.Address) == "" {
		return errors.New("stop address is required")
	}

	return p.Location.Validate()
}

func (s Stop) Props() StopProps {
	return s.props
}

type TrackingPoint struct {
	Location   geo.Location
	RecordedAt time.Time
}

func NewTrackingPoint(location geo.Location, at time.Time) (TrackingPoint, error) {
	p := TrackingPoint{
		Location:   location,
		RecordedAt: at,
	}

	return p, p.Validate()
}

func (p TrackingPoint) Validate() error {
	if p.RecordedAt.IsZero() {
		return errors.New("tracking timestamp is required")
	}

	return p.Location.Validate()
}

type StatusEntry struct {
	Status     Status
	OccurredAt time.Time
}

type CreateDeliveryProps struct {
	CourierID    ddd.ID
	OrderStoreID ddd.ID
	Pickup       Stop
	DropOff      Stop
	AssignedAt   time.Time
}

type DeliveryProps struct {
	CourierID    ddd.ID
	OrderStoreID ddd.ID
	Pickup       Stop
	DropOff      Stop
	Status       Status
	AssignedAt   time.Time
	History      []StatusEntry
	Track        []TrackingPoint
}

type Delivery struct {
	aggregate ddd.Aggregate[DeliveryProps]
}

func NewDelivery(p CreateDeliveryProps) (*Delivery, error) {
	props := DeliveryProps{
		CourierID:    p.CourierID,
		OrderStoreID: p.OrderStoreID,
		Pickup:       p.Pickup,
		DropOff:      p.DropOff,
		Status:       Assigned,
		AssignedAt:   p.AssignedAt,
		History: []StatusEntry{{
			Status:     Assigned,
			OccurredAt: p.AssignedAt,
		}},
	}
	a, err := ddd.NewAggregate(ddd.NewID(), props, ValidateDelivery)
	if err != nil {
		return nil, err
	}

	return &Delivery{aggregate: *a}, nil
}

func ValidateDelivery(p DeliveryProps) error {
	if p.CourierID.Empty() || p.OrderStoreID.Empty() || p.AssignedAt.IsZero() {
		return errors.New("delivery courier, store and assignment time are required")
	}

	if err := ValidateStop(p.Pickup.Props()); err != nil {
		return err
	}

	if err := ValidateStop(p.DropOff.Props()); err != nil {
		return err
	}

	switch p.Status {
	case Assigned, PickedUp, Delivered, Cancelled:
		return nil
	}

	return errors.New("invalid delivery status")
}

func (d *Delivery) ID() ddd.ID {
	return d.aggregate.ID()
}

func (d *Delivery) Version() uint64 {
	return d.aggregate.Version()
}

func (d *Delivery) Props() DeliveryProps {
	p := d.aggregate.Props()
	p.History = append([]StatusEntry(nil), p.History...)
	p.Track = append([]TrackingPoint(nil), p.Track...)

	return p
}

func (d *Delivery) Clone() *Delivery {
	c := *d

	return &c
}

func (d *Delivery) Active() bool {
	p := d.Props()

	return p.Status == Assigned || p.Status == PickedUp
}

func (d *Delivery) Transition(next Status, at time.Time) error {
	p := d.Props()
	valid := (p.Status == Assigned && (next == PickedUp || next == Cancelled)) || (p.Status == PickedUp && next == Delivered)
	if !valid {
		return errors.New("invalid delivery transition")
	}

	lastStatusAt := p.History[len(p.History)-1].OccurredAt
	beforeLastPosition := len(p.Track) > 0 && at.Before(p.Track[len(p.Track)-1].RecordedAt)
	if at.IsZero() || at.Before(lastStatusAt) || beforeLastPosition {
		return errors.New("delivery timestamps must be chronological")
	}

	p.Status = next
	p.History = append(p.History, StatusEntry{
		Status:     next,
		OccurredAt: at,
	})
	d.aggregate.Update(func(current *DeliveryProps) { *current = p })

	return nil
}

func (d *Delivery) RecordLocation(point TrackingPoint) error {
	if !d.Active() {
		return errors.New("cannot track an inactive delivery")
	}

	if err := point.Validate(); err != nil {
		return err
	}

	p := d.Props()
	lastStatusAt := p.History[len(p.History)-1].OccurredAt
	beforeLastPosition := len(p.Track) > 0 && point.RecordedAt.Before(p.Track[len(p.Track)-1].RecordedAt)
	if point.RecordedAt.Before(lastStatusAt) || beforeLastPosition {
		return errors.New("tracking timestamps must be chronological")
	}

	p.Track = append(p.Track, point)
	d.aggregate.Update(func(current *DeliveryProps) { *current = p })

	return nil
}
