package order_domain

import (
	"testing"
	"time"
)

func newDraft(t *testing.T) *OrderStore {
	t.Helper()
	s, err := NewOrderStore(CreateOrderStoreProps{
		OrderStoreId:    1,
		City:            "tehran",
		PreparationTime: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func populate(t *testing.T, s *OrderStore) {
	t.Helper()
	location, err := NewLocation(LocationProps{
		Latitude:  35,
		Longitude: 51,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, err := range []error{s.AddOrder(OrderProps{OrderId: "multi-store-order"}), s.AddCustomer(CustomerProps{
		Name:        "Customer",
		Mobile:      "09120000000",
		AddressText: "Destination",
		Location:    location,
	}), s.AddVendor(VendorProps{
		VendorId:    2,
		Title:       "Vendor",
		City:        "tehran",
		AddressText: "Pickup",
		Location:    location,
	}), s.AddItems(2, []ItemProps{{
		Title: "Item",
		Count: 1,
		Unit:  "piece",
	}})} {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestDraftAndLifecycle(t *testing.T) {
	s := newDraft(t)
	if s.ID().Empty() || s.Props().Status != CreateOrder {
		t.Fatal("invalid new store")
	}

	if err := s.MarkReady(time.Now()); err == nil {
		t.Fatal("incomplete store became ready")
	}

	populate(t, s)
	if err := s.Assign(time.Now()); err == nil {
		t.Fatal("draft assigned")
	}

	if err := s.MarkReady(time.Now()); err != nil {
		t.Fatal(err)
	}

	if err := s.AddItems(2, []ItemProps{{
		Title: "Late item",
		Count: 1,
		Unit:  "piece",
	}}); err == nil {
		t.Fatal("ready store was edited")
	}

	for _, transition := range []func(time.Time) error{s.Assign, s.PickUp, s.Complete} {
		if err := transition(time.Now()); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.Cancel(time.Now()); err == nil {
		t.Fatal("delivered store cancelled")
	}

	p := s.Props()
	if p.Status != Delivered || len(p.History) != 9 {
		t.Fatalf("unexpected history: %v", p.History)
	}
}

func TestItemsAreAtomicAndSnapshotsDoNotMutateStore(t *testing.T) {
	s := newDraft(t)
	populate(t, s)
	version := s.Version()
	before := len(s.Props().History)
	if err := s.AddItems(2, []ItemProps{
		{
			Title: "Valid",
			Count: 2,
			Unit:  "piece",
		},
		{
			Title: "Invalid",
			Count: 0,
			Unit:  "piece",
		},
	}); err == nil {
		t.Fatal("invalid item accepted")
	}

	if s.Version() != version || len(s.Props().History) != before || len(s.Props().Vendor.Props().Items) != 1 {
		t.Fatal("failed edit changed store")
	}

	p := s.Props()
	p.History[0] = OrderStatus{}
	vp := p.Vendor.Props()
	vp.Items[0] = Item{}
	if s.Props().History[0].ID().Empty() || s.Props().Vendor.Props().Items[0].ID().Empty() {
		t.Fatal("snapshot leaked mutable state")
	}

	clone := s.Clone()
	if err := clone.AddItems(2, []ItemProps{{
		Title: "Other",
		Count: 1,
		Unit:  "piece",
	}}); err != nil {
		t.Fatal(err)
	}

	if len(s.Props().Vendor.Props().Items) != 1 {
		t.Fatal("clone mutation leaked")
	}

	if err := s.AddItems(999, []ItemProps{{
		Title: "Wrong vendor",
		Count: 1,
		Unit:  "piece",
	}}); err == nil {
		t.Fatal("wrong vendor accepted")
	}
}

func TestModelValidation(t *testing.T) {
	if _, err := NewOrderStore(CreateOrderStoreProps{}); err == nil {
		t.Fatal("empty store accepted")
	}

	if _, err := NewOrder(OrderProps{}); err == nil {
		t.Fatal("empty order accepted")
	}

	if _, err := NewCustomer(CustomerProps{}); err == nil {
		t.Fatal("empty customer accepted")
	}

	if _, err := NewVendor(VendorProps{}); err == nil {
		t.Fatal("empty vendor accepted")
	}

	if _, err := NewItem(ItemProps{
		Title: "Item",
		Count: -1,
		Unit:  "piece",
	}); err == nil {
		t.Fatal("negative count accepted")
	}

	if _, err := NewOrderStatus(OrderStatusProps{
		Status:     "unknown",
		OccurredAt: time.Now(),
	}); err == nil {
		t.Fatal("unknown status accepted")
	}

	s := newDraft(t)
	populate(t, s)
	if err := s.MarkReady(time.Now().Add(-time.Hour)); err == nil {
		t.Fatal("backdated status accepted")
	}
}
