package memory_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	courier "github.com/calledchrist/courier-dispatch/internal/app/courier/domain"
	courierports "github.com/calledchrist/courier-dispatch/internal/app/courier/ports"
	domain "github.com/calledchrist/courier-dispatch/internal/app/dispatch/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/dispatch/ports"
	service "github.com/calledchrist/courier-dispatch/internal/app/dispatch/service"
	order "github.com/calledchrist/courier-dispatch/internal/app/order/domain"
	"github.com/calledchrist/courier-dispatch/internal/infrastructure/memory"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

func namedCourier(t *testing.T, storage *memory.Store, name string) *courier.Courier {
	t.Helper()
	couriers, err := storage.Couriers().ListByCity(context.Background(), "tehran")
	if err != nil {
		t.Fatal(err)
	}

	for _, candidate := range couriers {
		if candidate.Props().Name == name {
			return candidate
		}
	}

	t.Fatalf("courier %q missing", name)
	return nil
}

type noCourierPool struct {
	courierports.Repository
}

func (noCourierPool) ListByCity(context.Context, string) ([]*courier.Courier, error) {
	return nil, errors.New("manual assignment must not search the courier pool")
}

func TestManualAssignmentUsesRequestedCourier(t *testing.T) {
	storage, automatic, _, stores := setup(t)
	ctx := context.Background()
	selected := namedCourier(t, storage, "Ali")

	// Move Ali farther away so automatic dispatch prefers Sara for this store.
	location, err := courier.NewLocation(courier.LocationProps{Latitude: 35.69, Longitude: 51.4})
	if err != nil {
		t.Fatal(err)
	}

	err = storage.Do(ctx, func(tx context.Context) error {
		if err := selected.UpdateLocation(location, time.Now().UTC()); err != nil {
			return err
		}

		return storage.Couriers().Save(tx, selected)
	})
	if err != nil {
		t.Fatal(err)
	}

	deps := dependencies(storage)
	deps.Couriers = noCourierPool{storage.Couriers()}
	manual := service.NewManualAssignService(deps)
	result, err := manual.Assign(ctx, ports.ManualAssignRequest{
		OrderStoreID: stores[0].ID(),
		CourierID:    selected.ID(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.CourierID != selected.ID() || result.Method != domain.ManualAssignment {
		t.Fatalf("manual choice was not honored: %+v", result)
	}

	// A better-scoring courier was available, but the requested courier wins.
	competitor := namedCourier(t, storage, "Sara")
	storeProps := stores[0].Props()
	route, err := (memory.StraightLineRouter{}).Estimate(ctx, ports.RouteRequest{
		Origin:  competitor.Props().Location,
		Pickup:  storeProps.Vendor.Props().Location,
		DropOff: storeProps.Order.Props().Customer.Props().Location,
	})
	if err != nil {
		t.Fatal(err)
	}

	policy, err := domain.NewPolicy(domain.DefaultParameters())
	if err != nil {
		t.Fatal(err)
	}

	competitorScore, eligible := policy.Evaluate(
		domain.Requirements{City: "tehran"},
		domain.Candidate{
			CourierID:         competitor.ID(),
			City:              "tehran",
			Online:            true,
			Location:          competitor.Props().Location,
			LocationUpdatedAt: competitor.Props().LocationUpdatedAt,
			ActiveTripLimit:   competitor.Props().ActiveTripLimit,
		},
		route,
		time.Now().UTC(),
	)
	if !eligible || competitorScore.Score <= result.Score {
		t.Fatal("test requires a better-scoring alternative courier")
	}

	assignment, err := storage.Assignments().Get(ctx, result.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}

	props := assignment.Props()
	if props.Method != domain.ManualAssignment || props.Parameters != domain.DefaultParameters() || props.DeliveryID != result.DeliveryID {
		t.Fatalf("incorrect assignment record: %+v", props)
	}

	trip, err := storage.Deliveries().Get(ctx, result.DeliveryID)
	if err != nil {
		t.Fatal(err)
	}

	if trip.Props().CourierID != selected.ID() || trip.Props().OrderStoreID != stores[0].ID() {
		t.Fatal("delivery references do not match the manual choice")
	}

	saved, err := storage.Couriers().Get(ctx, selected.ID())
	if err != nil {
		t.Fatal(err)
	}

	if saved.ActiveTrips() != 1 || saved.Props().ActiveDeliveryIDs[0] != result.DeliveryID {
		t.Fatal("courier capacity was not reserved")
	}

	if _, err := automatic.Dispatch(ctx, request(stores[0])); err == nil {
		t.Fatal("automatic dispatch duplicated a manual assignment")
	}

	if _, err := manual.Assign(ctx, ports.ManualAssignRequest{OrderStoreID: stores[0].ID(), CourierID: selected.ID()}); err == nil {
		t.Fatal("manual assignment duplicated an existing delivery")
	}
}

func TestManualAssignmentRejectsIneligibleCourierWithoutFallback(t *testing.T) {
	tests := []struct {
		name        string
		courierName string
		storeIndex  int
		mutate      func(*courier.Courier) error
		parameters  *domain.Parameters
	}{
		{name: "offline", courierName: "Reza"},
		{name: "POS and car required", courierName: "Ali", storeIndex: 1},
		{
			name: "suspended", courierName: "Sara",
			mutate: func(c *courier.Courier) error { return c.ChangeStatus(courier.CourierSuspended) },
		},
		{
			name: "full capacity", courierName: "Sara",
			mutate: func(c *courier.Courier) error {
				if err := c.SetActiveTripLimit(1); err != nil {
					return err
				}

				return c.ReserveDelivery("existing-trip")
			},
		},
		{
			name: "pickup limit", courierName: "Sara",
			parameters: func() *domain.Parameters {
				p := domain.DefaultParameters()
				p.MaxPickupKM = 0.000001
				return &p
			}(),
		},
		{
			name: "route limit", courierName: "Sara",
			parameters: func() *domain.Parameters {
				p := domain.DefaultParameters()
				p.MaxRouteKM = 0.01
				return &p
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage, _, _, stores := setup(t)
			selected := namedCourier(t, storage, test.courierName)
			ctx := context.Background()

			if test.mutate != nil {
				err := storage.Do(ctx, func(tx context.Context) error {
					if err := test.mutate(selected); err != nil {
						return err
					}

					return storage.Couriers().Save(tx, selected)
				})
				if err != nil {
					t.Fatal(err)
				}
			}

			manual := service.NewManualAssignService(dependencies(storage))
			_, err := manual.Assign(ctx, ports.ManualAssignRequest{
				OrderStoreID: stores[test.storeIndex].ID(),
				CourierID:    selected.ID(),
				Parameters:   test.parameters,
			})
			if !errors.Is(err, domain.ErrCourierNotEligible) {
				t.Fatalf("expected ineligible error, got %v", err)
			}

			assertNoAssignment(t, storage, stores[test.storeIndex].ID())
		})
	}
}

func TestManualAssignmentValidatesReferences(t *testing.T) {
	storage, _, _, stores := setup(t)
	selected := namedCourier(t, storage, "Sara")
	manual := service.NewManualAssignService(dependencies(storage))

	for _, req := range []ports.ManualAssignRequest{
		{},
		{OrderStoreID: stores[0].ID()},
		{OrderStoreID: stores[0].ID(), CourierID: "missing"},
		{OrderStoreID: "missing", CourierID: selected.ID()},
		{OrderStoreID: stores[0].ID(), CourierID: selected.ID(), Parameters: &domain.Parameters{}},
	} {
		if _, err := manual.Assign(context.Background(), req); err == nil {
			t.Fatalf("invalid request accepted: %+v", req)
		}
	}

	assertNoAssignment(t, storage, stores[0].ID())
}

func assertNoAssignment(t *testing.T, storage *memory.Store, storeID ddd.ID) {
	t.Helper()
	ctx := context.Background()
	assignments, err := storage.Assignments().List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	active, err := storage.Deliveries().ActiveByStore(ctx, storeID)
	if err != nil {
		t.Fatal(err)
	}

	store, err := storage.Orders().Get(ctx, storeID)
	if err != nil {
		t.Fatal(err)
	}

	if len(assignments) != 0 || len(active) != 0 || store.Props().Status != order.OrderIsReady {
		t.Fatal("failed assignment changed persisted state")
	}
}

func TestManualAssignmentRollsBackFinalWriteFailure(t *testing.T) {
	storage, _, _, stores := setup(t)
	selected := namedCourier(t, storage, "Sara")
	deps := dependencies(storage)
	deps.Assignments = failingAssignments{storage.Assignments()}
	manual := service.NewManualAssignService(deps)

	result, err := manual.Assign(context.Background(), ports.ManualAssignRequest{
		OrderStoreID: stores[0].ID(),
		CourierID:    selected.ID(),
	})
	if !errors.Is(err, errInjected) || !result.AssignmentID.Empty() {
		t.Fatalf("unexpected failure result: %+v, %v", result, err)
	}

	assertNoAssignment(t, storage, stores[0].ID())
	saved, err := storage.Couriers().Get(context.Background(), selected.ID())
	if err != nil {
		t.Fatal(err)
	}

	if saved.ActiveTrips() != 0 || saved.Version() != selected.Version() {
		t.Fatal("courier reservation escaped rollback")
	}
}

func TestConcurrentAutomaticAndManualAssignment(t *testing.T) {
	storage, automatic, _, stores := setup(t)
	manual := service.NewManualAssignService(dependencies(storage))
	selected := namedCourier(t, storage, "Sara")
	var successes atomic.Int32
	var workers sync.WaitGroup

	for i := range 12 {
		workers.Go(func() {
			var err error
			if i%2 == 0 {
				_, err = automatic.Dispatch(context.Background(), request(stores[0]))
			} else {
				_, err = manual.Assign(context.Background(), ports.ManualAssignRequest{
					OrderStoreID: stores[0].ID(),
					CourierID:    selected.ID(),
				})
			}

			if err == nil {
				successes.Add(1)
			}
		})
	}

	workers.Wait()
	if successes.Load() != 1 {
		t.Fatalf("expected one assignment, got %d", successes.Load())
	}
}

func TestManualAssignmentRequiresSameCityAndRecentLocation(t *testing.T) {
	tests := []struct {
		name        string
		city        string
		hasLocation bool
		age         time.Duration
	}{
		{name: "other city", city: "shiraz", hasLocation: true},
		{name: "missing location", city: "tehran"},
		{name: "stale location", city: "tehran", hasLocation: true, age: 6 * time.Minute},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage, _, _, stores := setup(t)
			selected, err := courier.NewCourier(courier.CreateCourierProps{
				Name:         "Manual candidate",
				Mobile:       "09120000008",
				City:         test.city,
				Status:       courier.CourierOnline,
				VehicleType:  courier.VehicleCar,
				HasPOSDevice: true,
			})
			if err != nil {
				t.Fatal(err)
			}

			if test.hasLocation {
				location := stores[0].Props().Vendor.Props().Location
				if err := selected.UpdateLocation(location, time.Now().Add(-test.age)); err != nil {
					t.Fatal(err)
				}
			}

			ctx := context.Background()
			err = storage.Do(ctx, func(tx context.Context) error {
				return storage.Couriers().Save(tx, selected)
			})
			if err != nil {
				t.Fatal(err)
			}

			manual := service.NewManualAssignService(dependencies(storage))
			_, err = manual.Assign(ctx, ports.ManualAssignRequest{
				OrderStoreID: stores[0].ID(),
				CourierID:    selected.ID(),
			})
			if !errors.Is(err, domain.ErrCourierNotEligible) {
				t.Fatalf("expected ineligible error, got %v", err)
			}

			assertNoAssignment(t, storage, stores[0].ID())
		})
	}
}
