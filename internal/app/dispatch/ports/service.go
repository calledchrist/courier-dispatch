package dispatch_ports

import (
	"context"
	"time"

	domain "github.com/calledchrist/courier-dispatch/internal/app/dispatch/domain"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
	"github.com/calledchrist/courier-dispatch/internal/shared/geo"
)

type DispatchRequest struct {
	OrderStoreID ddd.ID
	City         string
	Parameters   domain.Parameters
}

type ManualAssignRequest struct {
	OrderStoreID ddd.ID
	CourierID    ddd.ID
	// Nil uses the default policy. City is derived from the order store.
	Parameters *domain.Parameters
}

type DispatchResult struct {
	AssignmentID ddd.ID
	DeliveryID   ddd.ID
	CourierID    ddd.ID
	OrderStoreID ddd.ID
	Method       domain.AssignmentMethod
	Score        float64
	Route        domain.RouteEstimate
}

type Service interface {
	Dispatch(context.Context, DispatchRequest) (DispatchResult, error)
}

type ManualAssignmentService interface {
	Assign(context.Context, ManualAssignRequest) (DispatchResult, error)
}

// RouteRequest describes remaining stops in their planned order followed by the new work.
type RouteRequest struct {
	Origin         geo.Location
	Pickup         geo.Location
	DropOff        geo.Location
	RemainingStops []geo.Location
}

type Router interface {
	Estimate(context.Context, RouteRequest) (domain.RouteEstimate, error)
}

type Clock interface {
	Now() time.Time
}
