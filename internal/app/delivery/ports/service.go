package delivery_ports

import (
	"context"
	"time"

	domain "github.com/calledchrist/courier-dispatch/internal/app/delivery/domain"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
	"github.com/calledchrist/courier-dispatch/internal/shared/geo"
)

type DeliveryServiceInboundPort interface {
	Transition(context.Context, ddd.ID, domain.Status, time.Time) error
	TrackCourier(context.Context, ddd.ID, geo.Location, time.Time) error
	ActiveDeliveries(context.Context, ddd.ID) ([]*domain.Delivery, error)
}
