package courier_domain

import (
	"testing"
	"time"
)

func TestCourierCapacityAndLocation(t *testing.T) {
	c, err := NewCourier(CreateCourierProps{
		Name:            "Courier",
		Mobile:          "09120000000",
		City:            "tehran",
		VehicleType:     VehicleCar,
		Status:          CourierOnline,
		HasPOSDevice:    true,
		ActiveTripLimit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !c.Props().HasPOSDevice {
		t.Fatal("constructor discarded POS")
	}

	if err := c.ReserveDelivery("trip-1"); err == nil {
		t.Fatal("courier without location accepted")
	}

	location, err := NewLocation(LocationProps{})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	if err := c.UpdateLocation(location, now); err != nil {
		t.Fatal(err)
	}

	if err := c.UpdateLocation(location, now.Add(-time.Second)); err == nil {
		t.Fatal("backdated location accepted")
	}

	if err := c.ReserveDelivery("trip-1"); err != nil {
		t.Fatal(err)
	}

	if err := c.ReserveDelivery("trip-2"); err == nil {
		t.Fatal("capacity exceeded")
	}

	if err := c.SetActiveTripLimit(0); err == nil {
		t.Fatal("invalid limit accepted")
	}

	if err := c.ReleaseDelivery("other"); err == nil {
		t.Fatal("unreserved trip released")
	}

	snapshot := c.Props()
	snapshot.ActiveDeliveryIDs[0] = "other"
	if c.Props().ActiveDeliveryIDs[0] != "trip-1" {
		t.Fatal("mutable snapshot")
	}

	if err := c.ReleaseDelivery("trip-1"); err != nil {
		t.Fatal(err)
	}

	if !c.Available() {
		t.Fatal("capacity not released")
	}

	if err := c.ChangeStatus(CourierSuspended); err != nil {
		t.Fatal(err)
	}

	if err := c.ReserveDelivery("trip-2"); err == nil {
		t.Fatal("suspended courier accepted")
	}
}
