package memory_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	courier "github.com/calledchrist/courier-dispatch/internal/app/courier/domain"
	delivery "github.com/calledchrist/courier-dispatch/internal/app/delivery/domain"
	deliveryservice "github.com/calledchrist/courier-dispatch/internal/app/delivery/service"
	domain "github.com/calledchrist/courier-dispatch/internal/app/dispatch/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/dispatch/ports"
	service "github.com/calledchrist/courier-dispatch/internal/app/dispatch/service"
	order "github.com/calledchrist/courier-dispatch/internal/app/order/domain"
	"github.com/calledchrist/courier-dispatch/internal/infrastructure/memory"
)

func setup(t *testing.T) (*memory.Store, *service.Service, *deliveryservice.DeliveryService, []*order.OrderStore) {
	t.Helper()
	db := memory.New()
	if err := memory.SeedDemo(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	dispatch := service.New(dependencies(db))
	lifecycle := deliveryservice.NewDeliveryService(deliveryservice.Dependencies{
		UOW:        db,
		Orders:     db.Orders(),
		Couriers:   db.Couriers(),
		Deliveries: db.Deliveries(),
	})
	stores, err := db.Orders().List(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return db, dispatch, lifecycle, stores
}

func dependencies(db *memory.Store) service.Dependencies {
	return service.Dependencies{
		UOW:         db,
		Orders:      db.Orders(),
		Couriers:    db.Couriers(),
		Deliveries:  db.Deliveries(),
		Assignments: db.Assignments(),
		Router:      memory.StraightLineRouter{},
		Clock:       memory.Clock{},
	}
}

func request(s *order.OrderStore) ports.DispatchRequest {
	return ports.DispatchRequest{
		OrderStoreID: s.ID(),
		City:         "tehran",
		Parameters:   domain.DefaultParameters(),
	}
}

func TestDispatchAndLifecycle(t *testing.T) {
	db, dispatch, lifecycle, stores := setup(t)
	ctx := context.Background()
	if stores[0].Props().Order.Props().OrderId != stores[1].Props().Order.Props().OrderId {
		t.Fatal("fixture must represent a multi-store order")
	}

	results := []ports.DispatchResult{}
	for _, store := range stores {
		result, err := dispatch.Dispatch(ctx, request(store))
		if err != nil {
			t.Fatal(err)
		}

		results = append(results, result)
		c, err := db.Couriers().Get(ctx, result.CourierID)
		if err != nil {
			t.Fatal(err)
		}

		if store.Props().RequiresPOS && (!c.Props().HasPOSDevice || c.Props().VehicleType != courier.VehicleCar) {
			t.Fatal("requirements ignored")
		}
	}

	if results[0].DeliveryID == results[1].DeliveryID {
		t.Fatal("stores share a delivery")
	}

	result := results[0]
	if _, err := dispatch.Dispatch(ctx, request(stores[0])); err == nil {
		t.Fatal("duplicate assignment succeeded")
	}

	trip, err := db.Deliveries().Get(ctx, result.DeliveryID)
	if err != nil {
		t.Fatal(err)
	}

	c, err := db.Couriers().Get(ctx, result.CourierID)
	if err != nil {
		t.Fatal(err)
	}

	before := c.ActiveTrips()
	if err := lifecycle.Transition(ctx, trip.ID(), delivery.Delivered, time.Now()); err == nil {
		t.Fatal("delivery completed before pickup")
	}

	if err := lifecycle.TrackCourier(ctx, c.ID(), c.Props().Location, time.Now()); err != nil {
		t.Fatal(err)
	}

	if err := lifecycle.Transition(ctx, trip.ID(), delivery.PickedUp, time.Now()); err != nil {
		t.Fatal(err)
	}

	if err := lifecycle.Transition(ctx, trip.ID(), delivery.Delivered, time.Now()); err != nil {
		t.Fatal(err)
	}

	c, err = db.Couriers().Get(ctx, c.ID())
	if err != nil {
		t.Fatal(err)
	}

	if c.ActiveTrips() != before-1 {
		t.Fatal("capacity not released")
	}

	trip, err = db.Deliveries().Get(ctx, trip.ID())
	if err != nil {
		t.Fatal(err)
	}

	if len(trip.Props().Track) != 1 || trip.Active() {
		t.Fatal("tracking/lifecycle mismatch")
	}

	if err := lifecycle.Transition(ctx, trip.ID(), delivery.Delivered, time.Now()); err == nil {
		t.Fatal("double completion succeeded")
	}

	sibling, err := db.Orders().Get(ctx, stores[1].ID())
	if err != nil {
		t.Fatal(err)
	}

	if sibling.Props().Status != order.Assigned {
		t.Fatal("sibling store changed")
	}

	if err := lifecycle.Transition(ctx, results[1].DeliveryID, delivery.Cancelled, time.Now()); err != nil {
		t.Fatal(err)
	}
}

type failingAssignments struct {
	ports.Repository
}

var errInjected = errors.New("injected final write failure")

func (f failingAssignments) Save(context.Context, *domain.Assignment) error {
	return errInjected
}

func TestAssignmentFailureRollsBackEveryAggregate(t *testing.T) {
	db, _, _, stores := setup(t)
	deps := dependencies(db)
	deps.Assignments = failingAssignments{db.Assignments()}
	dispatch := service.New(deps)
	ctx := context.Background()
	result, err := dispatch.Dispatch(ctx, request(stores[0]))
	if !errors.Is(err, errInjected) {
		t.Fatalf("wrong error: %v", err)
	}

	if !result.AssignmentID.Empty() {
		t.Fatal("failed transaction returned assignment")
	}

	store, err := db.Orders().Get(ctx, stores[0].ID())
	if err != nil {
		t.Fatal(err)
	}

	if store.Props().Status != order.OrderIsReady || store.Version() != stores[0].Version() {
		t.Fatal("store not rolled back")
	}

	couriers, err := db.Couriers().ListByCity(ctx, "tehran")
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range couriers {
		if c.ActiveTrips() != 0 {
			t.Fatal("reservation not rolled back")
		}
	}

	active, err := db.Deliveries().ActiveByStore(ctx, store.ID())
	if err != nil {
		t.Fatal(err)
	}

	assignments, err := db.Assignments().List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(active) != 0 || len(assignments) != 0 {
		t.Fatal("failed transaction persisted work")
	}
}

func TestConcurrentDispatchCannotDoubleAssign(t *testing.T) {
	db, dispatch, _, stores := setup(t)
	var success atomic.Int32
	var wg sync.WaitGroup
	for range 12 {
		wg.Go(func() {
			if _, err := dispatch.Dispatch(context.Background(), request(stores[0])); err == nil {
				success.Add(1)
			}
		})
	}

	wg.Wait()
	if success.Load() != 1 {
		t.Fatalf("%d successful assignments", success.Load())
	}

	a, err := db.Assignments().List(context.Background())
	if err != nil || len(a) != 1 {
		t.Fatal("assignment count", len(a), err)
	}
}

func TestConcurrentStoresRespectOneRemainingSlot(t *testing.T) {
	db, dispatch, _, stores := setup(t)
	ctx := context.Background()
	err := db.Do(ctx, func(tx context.Context) error {
		couriers, err := db.Couriers().ListByCity(tx, "tehran")
		if err != nil {
			return err
		}

		for _, c := range couriers {
			if c.Props().Name == "Sara" {
				if err := c.SetActiveTripLimit(1); err != nil {
					return err
				}
			} else {
				if err := c.ChangeStatus(courier.CourierOffline); err != nil {
					return err
				}
			}

			if err := db.Couriers().Save(tx, c); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var success atomic.Int32
	var wg sync.WaitGroup
	for _, store := range stores {
		wg.Go(func() {
			if _, err := dispatch.Dispatch(ctx, request(store)); err == nil {
				success.Add(1)
			}
		})
	}

	wg.Wait()
	if success.Load() != 1 {
		t.Fatalf("capacity allowed %d assignments", success.Load())
	}
}

func TestNoEligibleCourierDoesNotWrite(t *testing.T) {
	db, dispatch, _, stores := setup(t)
	ctx := context.Background()
	req := request(stores[0])
	req.Parameters.MaxPickupKM = 0.000001
	if _, err := dispatch.Dispatch(ctx, req); !errors.Is(err, domain.ErrNoEligibleCourier) {
		t.Fatalf("expected no candidate, got %v", err)
	}

	assignments, err := db.Assignments().List(ctx)
	if err != nil || len(assignments) != 0 {
		t.Fatal("unexpected write")
	}

	req = request(stores[0])
	req.City = "other"
	if _, err := dispatch.Dispatch(ctx, req); err == nil {
		t.Fatal("wrong city accepted")
	}
}

func TestTransactionIsolationCancellationAndPanic(t *testing.T) {
	db, _, _, stores := setup(t)
	ctx := context.Background()
	id := stores[0].ID()
	original := stores[0].Version()
	if err := db.Orders().Save(ctx, stores[0]); !errors.Is(err, memory.ErrTransactionRequired) {
		t.Fatal("write outside UOW accepted")
	}

	cancelled, cancel := context.WithCancel(ctx)
	var retained context.Context
	err := db.Do(cancelled, func(tx context.Context) error {
		retained = tx
		store, err := db.Orders().Get(tx, id)
		if err != nil {
			return err
		}

		if err := store.Assign(time.Now()); err != nil {
			return err
		}

		if err := db.Orders().Save(tx, store); err != nil {
			return err
		}

		cancel()

		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}

	if _, err := db.Orders().Get(retained, id); err == nil {
		t.Fatal("retained transaction usable")
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected panic")
			}
		}()
		_ = db.Do(ctx, func(tx context.Context) error {
			s, err := db.Orders().Get(tx, id)
			if err != nil {
				return err
			}

			if err := s.Assign(time.Now()); err != nil {
				return err
			}

			if err := db.Orders().Save(tx, s); err != nil {
				return err
			}

			panic("rollback")
		})
	}()
	s, err := db.Orders().Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	if s.Version() != original || s.Props().Status != order.OrderIsReady {
		t.Fatal("rollback leaked state")
	}

	if err := s.Assign(time.Now()); err != nil {
		t.Fatal(err)
	}

	again, err := db.Orders().Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	if again.Props().Status != order.OrderIsReady {
		t.Fatal("repository get leaked reference")
	}
}

func TestRouteIncludesExistingStops(t *testing.T) {
	location, _ := courier.NewLocation(courier.LocationProps{})
	far, _ := courier.NewLocation(courier.LocationProps{Longitude: 0.1})
	router := memory.StraightLineRouter{}
	estimate, err := router.Estimate(context.Background(), ports.RouteRequest{
		Origin:         location,
		Pickup:         location,
		DropOff:        location,
		RemainingStops: []courier.Location{far},
	})
	if err != nil {
		t.Fatal(err)
	}

	if estimate.TotalRouteKM < 22 || estimate.AdditionalKM < 11 {
		t.Fatal("existing route omitted", estimate)
	}
}

func TestTrackingFailureRollsBackCourierLocation(t *testing.T) {
	db, dispatch, lifecycle, stores := setup(t)
	ctx := context.Background()
	result, err := dispatch.Dispatch(ctx, request(stores[0]))
	if err != nil {
		t.Fatal(err)
	}

	c, err := db.Couriers().Get(ctx, result.CourierID)
	if err != nil {
		t.Fatal(err)
	}

	trip, err := db.Deliveries().Get(ctx, result.DeliveryID)
	if err != nil {
		t.Fatal(err)
	}

	futurePickup := time.Now().Add(time.Minute)
	if err := lifecycle.Transition(ctx, trip.ID(), delivery.PickedUp, futurePickup); err != nil {
		t.Fatal(err)
	}
	// The position is newer than the courier's latest position but older than
	// the delivery's pickup: failure must roll back the courier update too.

	if err := lifecycle.TrackCourier(ctx, c.ID(), c.Props().Location, futurePickup.Add(-time.Second)); err == nil {
		t.Fatal("invalid tracking accepted")
	}

	after, err := db.Couriers().Get(ctx, c.ID())
	if err != nil {
		t.Fatal(err)
	}

	if !after.Props().LocationUpdatedAt.Equal(c.Props().LocationUpdatedAt) || after.Version() != c.Version() {
		t.Fatal("courier update escaped rollback")
	}
}
