package service

import (
	"context"
	"time"

	domain "github.com/calledchrist/courier-dispatch/internal/app/delivery/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/delivery/ports"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
	"github.com/calledchrist/courier-dispatch/internal/shared/geo"
)

type DeliveryService struct {
	deps Dependencies
}

func NewDeliveryService(deps Dependencies) *DeliveryService {
	return &DeliveryService{deps: deps}
}

var _ ports.DeliveryServiceInboundPort = (*DeliveryService)(nil)

func (s *DeliveryService) ActiveDeliveries(ctx context.Context, courierID ddd.ID) ([]*domain.Delivery, error) {
	return s.deps.Deliveries.ActiveByCourier(ctx, courierID)
}

func (s *DeliveryService) Transition(ctx context.Context, id ddd.ID, next domain.Status, at time.Time) error {
	return s.deps.UOW.Do(ctx, func(tx context.Context) error {
		trip, err := s.deps.Deliveries.Get(tx, id)
		if err != nil {
			return err
		}

		if err := trip.Transition(next, at); err != nil {
			return err
		}

		store, err := s.deps.Orders.Get(tx, trip.Props().OrderStoreID)
		if err != nil {
			return err
		}

		switch next {
		case domain.PickedUp:
			err = store.PickUp(at)
		case domain.Delivered:
			err = store.Complete(at)
		case domain.Cancelled:
			err = store.Cancel(at)
		}

		if err != nil {
			return err
		}

		if !trip.Active() {
			c, err := s.deps.Couriers.Get(tx, trip.Props().CourierID)
			if err != nil {
				return err
			}

			if err := c.ReleaseDelivery(trip.ID()); err != nil {
				return err
			}

			if err := s.deps.Couriers.Save(tx, c); err != nil {
				return err
			}
		}

		if err := s.deps.Deliveries.Save(tx, trip); err != nil {
			return err
		}

		return s.deps.Orders.Save(tx, store)
	})
}

func (s *DeliveryService) TrackCourier(ctx context.Context, id ddd.ID, location geo.Location, at time.Time) error {
	point, err := domain.NewTrackingPoint(location, at)
	if err != nil {
		return err
	}

	return s.deps.UOW.Do(ctx, func(tx context.Context) error {
		c, err := s.deps.Couriers.Get(tx, id)
		if err != nil {
			return err
		}

		if err := c.UpdateLocation(location, at); err != nil {
			return err
		}

		active, err := s.deps.Deliveries.ActiveByCourier(tx, id)
		if err != nil {
			return err
		}

		for _, trip := range active {
			if err := trip.RecordLocation(point); err != nil {
				return err
			}

			if err := s.deps.Deliveries.Save(tx, trip); err != nil {
				return err
			}
		}

		return s.deps.Couriers.Save(tx, c)
	})
}
