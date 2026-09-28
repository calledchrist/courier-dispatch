package memory

import (
	"context"
	"time"

	domain "github.com/calledchrist/courier-dispatch/internal/app/dispatch/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/dispatch/ports"
	"github.com/calledchrist/courier-dispatch/internal/shared/geo"
)

type Clock struct{}

func (Clock) Now() time.Time {
	return time.Now().UTC()
}

// StraightLineRouter is a demo approximation, not a road-network routing engine.
// Existing work is completed in assignment order, then the new pickup/drop-off.
type StraightLineRouter struct{}

var _ ports.Router = StraightLineRouter{}

func (StraightLineRouter) Estimate(ctx context.Context, r ports.RouteRequest) (domain.RouteEstimate, error) {
	if err := ctx.Err(); err != nil {
		return domain.RouteEstimate{}, err
	}

	locations := append([]geo.Location{r.Origin, r.Pickup, r.DropOff}, r.RemainingStops...)
	for _, l := range locations {
		if err := l.Validate(); err != nil {
			return domain.RouteEstimate{}, err
		}
	}

	current := r.Origin
	existing := 0.0
	for _, stop := range r.RemainingStops {
		existing += current.DistanceKM(stop)
		current = stop
	}

	additional := current.DistanceKM(r.Pickup) + r.Pickup.DistanceKM(r.DropOff)

	return domain.RouteEstimate{
		PickupKM:     r.Origin.DistanceKM(r.Pickup),
		TotalRouteKM: existing + additional,
		AdditionalKM: additional,
	}, nil
}
