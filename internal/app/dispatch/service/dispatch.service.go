package dispatch_service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	courier "github.com/calledchrist/courier-dispatch/internal/app/courier/domain"
	courierports "github.com/calledchrist/courier-dispatch/internal/app/courier/ports"
	delivery "github.com/calledchrist/courier-dispatch/internal/app/delivery/domain"
	deliveryports "github.com/calledchrist/courier-dispatch/internal/app/delivery/ports"
	domain "github.com/calledchrist/courier-dispatch/internal/app/dispatch/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/dispatch/ports"
	order "github.com/calledchrist/courier-dispatch/internal/app/order/domain"
	orderports "github.com/calledchrist/courier-dispatch/internal/app/order/ports"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
	"github.com/calledchrist/courier-dispatch/internal/shared/uow"
)

type Dependencies struct {
	UOW         uow.UnitOfWork
	Orders      orderports.Repository
	Couriers    courierports.Repository
	Deliveries  deliveryports.Repository
	Assignments ports.Repository
	Router      ports.Router
	Clock       ports.Clock
}

type Service struct {
	deps Dependencies
}

func New(deps Dependencies) *Service {
	return &Service{deps: deps}
}

var _ ports.Service = (*Service)(nil)

func (s *Service) Dispatch(ctx context.Context, req ports.DispatchRequest) (ports.DispatchResult, error) {
	if req.OrderStoreID.Empty() || strings.TrimSpace(req.City) == "" {
		return ports.DispatchResult{}, errors.New("store ID and selected city are required")
	}

	policy, err := domain.NewPolicy(req.Parameters)
	if err != nil {
		return ports.DispatchResult{}, err
	}

	var result ports.DispatchResult
	err = s.deps.UOW.Do(ctx, func(tx context.Context) error {
		store, err := s.assignableStore(tx, req.OrderStoreID)
		if err != nil {
			return err
		}

		if store.Props().City != req.City {
			return errors.New("store does not belong to selected city")
		}

		couriers, err := s.deps.Couriers.ListByCity(tx, req.City)
		if err != nil {
			return err
		}

		now := s.deps.Clock.Now()
		eligible := make(map[ddd.ID]*courier.Courier)
		var scored []domain.ScoredCandidate

		for _, candidate := range couriers {
			score, accepted, err := s.evaluateCourier(tx, store, candidate, policy, now)
			if err != nil {
				return err
			}

			if accepted {
				scored = append(scored, score)
				eligible[candidate.ID()] = candidate
			}
		}

		ranked := domain.Rank(scored)
		if len(ranked) == 0 {
			return domain.ErrNoEligibleCourier
		}

		best := ranked[0]
		result, err = s.assign(tx, store, eligible[best.CourierID], best, req.Parameters, now, domain.AutomaticAssignment)

		return err
	})
	if err != nil {
		return ports.DispatchResult{}, err
	}

	return result, nil
}

// assignableStore is shared by automatic and manual assignment. Its caller must
// use the transaction context for the entire read, evaluate, and write workflow.
func (s *Service) assignableStore(ctx context.Context, id ddd.ID) (*order.OrderStore, error) {
	store, err := s.deps.Orders.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if store.Props().Status != order.OrderIsReady {
		return nil, errors.New("store is not ready for dispatch")
	}

	active, err := s.deps.Deliveries.ActiveByStore(ctx, store.ID())
	if err != nil {
		return nil, err
	}

	if len(active) > 0 {
		return nil, errors.New("store already has an active delivery")
	}

	return store, nil
}

func (s *Service) evaluateCourier(
	ctx context.Context,
	store *order.OrderStore,
	candidate *courier.Courier,
	policy domain.Policy,
	now time.Time,
) (domain.ScoredCandidate, bool, error) {
	storeProps := store.Props()
	courierProps := candidate.Props()
	requirements := domain.Requirements{
		City:        storeProps.City,
		RequiresPOS: storeProps.RequiresPOS,
		RequiresCar: storeProps.VehicleNeeded == order.VehicleCar,
	}
	snapshot := domain.Candidate{
		CourierID:         candidate.ID(),
		City:              courierProps.City,
		Online:            courierProps.Status == courier.CourierOnline,
		HasPOSDevice:      courierProps.HasPOSDevice,
		HasCar:            courierProps.VehicleType == courier.VehicleCar,
		Location:          courierProps.Location,
		LocationUpdatedAt: courierProps.LocationUpdatedAt,
		ActiveTrips:       candidate.ActiveTrips(),
		ActiveTripLimit:   courierProps.ActiveTripLimit,
	}
	if !policy.Eligible(requirements, snapshot, now) {
		return domain.ScoredCandidate{}, false, nil
	}

	deliveries, err := s.deps.Deliveries.ActiveByCourier(ctx, candidate.ID())
	if err != nil {
		return domain.ScoredCandidate{}, false, err
	}

	if len(deliveries) != candidate.ActiveTrips() {
		return domain.ScoredCandidate{}, false, errors.New("courier reservations and active deliveries disagree")
	}

	routeRequest := ports.RouteRequest{
		Origin:  courierProps.Location,
		Pickup:  storeProps.Vendor.Props().Location,
		DropOff: storeProps.Order.Props().Customer.Props().Location,
	}
	reserved := make(map[ddd.ID]bool)
	for _, id := range courierProps.ActiveDeliveryIDs {
		reserved[id] = true
	}

	for _, trip := range deliveries {
		if !reserved[trip.ID()] {
			return domain.ScoredCandidate{}, false, errors.New("courier reservation references disagree")
		}

		props := trip.Props()
		if props.Status == delivery.Assigned {
			routeRequest.RemainingStops = append(routeRequest.RemainingStops, props.Pickup.Props().Location)
		}

		routeRequest.RemainingStops = append(routeRequest.RemainingStops, props.DropOff.Props().Location)
	}

	estimate, err := s.deps.Router.Estimate(ctx, routeRequest)
	if err != nil {
		return domain.ScoredCandidate{}, false, fmt.Errorf("estimate route: %w", err)
	}

	if err := estimate.Validate(); err != nil {
		return domain.ScoredCandidate{}, false, err
	}

	score, accepted := policy.Evaluate(requirements, snapshot, estimate, now)

	return score, accepted, nil
}

// assign persists all four aggregates using the caller's unit of work.
func (s *Service) assign(
	ctx context.Context,
	store *order.OrderStore,
	selected *courier.Courier,
	score domain.ScoredCandidate,
	parameters domain.Parameters,
	at time.Time,
	method domain.AssignmentMethod,
) (ports.DispatchResult, error) {
	storeProps := store.Props()
	pickup, err := delivery.NewStop(delivery.StopProps{
		Address:  storeProps.Vendor.Props().AddressText,
		Location: storeProps.Vendor.Props().Location,
	})
	if err != nil {
		return ports.DispatchResult{}, err
	}

	dropOff, err := delivery.NewStop(delivery.StopProps{
		Address:  storeProps.Order.Props().Customer.Props().AddressText,
		Location: storeProps.Order.Props().Customer.Props().Location,
	})
	if err != nil {
		return ports.DispatchResult{}, err
	}

	trip, err := delivery.NewDelivery(delivery.CreateDeliveryProps{
		CourierID:    selected.ID(),
		OrderStoreID: store.ID(),
		Pickup:       pickup,
		DropOff:      dropOff,
		AssignedAt:   at,
	})
	if err != nil {
		return ports.DispatchResult{}, err
	}

	assignment, err := domain.NewAssignment(domain.AssignmentProps{
		CourierID:    selected.ID(),
		OrderStoreID: store.ID(),
		DeliveryID:   trip.ID(),
		Method:       method,
		Score:        score.Score,
		Route:        score.Route,
		Parameters:   parameters,
		AssignedAt:   at,
	})
	if err != nil {
		return ports.DispatchResult{}, err
	}

	if err := selected.ReserveDelivery(trip.ID()); err != nil {
		return ports.DispatchResult{}, err
	}

	if err := store.Assign(at); err != nil {
		return ports.DispatchResult{}, err
	}

	if err := s.deps.Couriers.Save(ctx, selected); err != nil {
		return ports.DispatchResult{}, err
	}

	if err := s.deps.Orders.Save(ctx, store); err != nil {
		return ports.DispatchResult{}, err
	}

	if err := s.deps.Deliveries.Save(ctx, trip); err != nil {
		return ports.DispatchResult{}, err
	}

	if err := s.deps.Assignments.Save(ctx, assignment); err != nil {
		return ports.DispatchResult{}, err
	}

	return ports.DispatchResult{
		AssignmentID: assignment.ID(),
		DeliveryID:   trip.ID(),
		CourierID:    selected.ID(),
		OrderStoreID: store.ID(),
		Method:       method,
		Score:        score.Score,
		Route:        score.Route,
	}, nil
}
