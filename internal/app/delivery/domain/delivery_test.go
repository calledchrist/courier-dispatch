package delivery_domain

import (
	"testing"
	"time"

	"github.com/calledchrist/courier-dispatch/internal/shared/geo"
)

func TestDeliveryValidationAndTracking(t *testing.T) {
	if _, err := NewDelivery(CreateDeliveryProps{}); err == nil {
		t.Fatal("missing references accepted")
	}

	if _, err := NewStop(StopProps{}); err == nil {
		t.Fatal("missing stop accepted")
	}

	location, err := geo.NewLocation(geo.LocationProps{})
	if err != nil {
		t.Fatal(err)
	}

	stop, err := NewStop(StopProps{
		Address:  "Address",
		Location: location,
	})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	trip, err := NewDelivery(CreateDeliveryProps{
		CourierID:    "courier",
		OrderStoreID: "store",
		Pickup:       stop,
		DropOff:      stop,
		AssignedAt:   now,
	})
	if err != nil {
		t.Fatal(err)
	}

	point, err := NewTrackingPoint(location, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	if err := trip.RecordLocation(point); err != nil {
		t.Fatal(err)
	}

	if err := trip.Transition(PickedUp, now); err == nil {
		t.Fatal("transition before latest position accepted")
	}

	if err := trip.Transition(PickedUp, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}

	if err := trip.Transition(Cancelled, now.Add(3*time.Second)); err == nil {
		t.Fatal("cancelled after pickup")
	}

	if err := trip.RecordLocation(point); err == nil {
		t.Fatal("position before pickup accepted")
	}

	snapshot := trip.Props()
	snapshot.Track[0] = TrackingPoint{}
	snapshot.History[0] = StatusEntry{}
	if trip.Props().Track[0].RecordedAt.IsZero() || trip.Props().History[0].OccurredAt.IsZero() {
		t.Fatal("mutable snapshot leaked")
	}

	if err := trip.Transition(Delivered, now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}

	point.RecordedAt = now.Add(4 * time.Second)
	if err := trip.RecordLocation(point); err == nil {
		t.Fatal("terminal delivery tracked")
	}
}
