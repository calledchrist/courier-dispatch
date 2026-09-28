package order_domain

import (
	"errors"
	"strings"
	"time"

	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

type OrderStore struct {
	aggregate ddd.Aggregate[OrderStoreProps]
}

func NewOrderStore(p CreateOrderStoreProps) (*OrderStore, error) {
	props := OrderStoreProps{
		OrderStoreId:    p.OrderStoreId,
		City:            p.City,
		VehicleNeeded:   p.VehicleNeeded,
		IsCash:          p.IsCash,
		RequiresPOS:     p.RequiresPOS,
		PreparationTime: p.PreparationTime,
		Status:          CreateOrder,
	}
	if err := ValidateOrderStore(props); err != nil {
		return nil, err
	}

	status, err := NewOrderStatus(OrderStatusProps{
		Status:     CreateOrder,
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	props.History = []OrderStatus{status}
	a, err := ddd.NewAggregate(ddd.NewID(), props, ValidateOrderStore)
	if err != nil {
		return nil, err
	}

	return &OrderStore{aggregate: *a}, nil
}

// ValidateOrderStore validates the draft. MarkReady enforces completeness.
func ValidateOrderStore(p OrderStoreProps) error {
	if p.OrderStoreId == 0 || strings.TrimSpace(p.City) == "" || p.PreparationTime.IsZero() {
		return errors.New("store ID, city and preparation time are required")
	}

	if p.VehicleNeeded != "" && p.VehicleNeeded != VehicleBike && p.VehicleNeeded != VehicleCar {
		return errors.New("invalid required vehicle")
	}

	return nil
}

func (s *OrderStore) ID() ddd.ID {
	return s.aggregate.ID()
}

func (s *OrderStore) Version() uint64 {
	return s.aggregate.Version()
}

func (s *OrderStore) Props() OrderStoreProps {
	p := s.aggregate.Props()
	p.Vendor.entity.Update(func(v *VendorProps) { v.Items = append([]Item(nil), v.Items...) })
	p.History = append([]OrderStatus(nil), p.History...)

	return p
}

func (s *OrderStore) Clone() *OrderStore {
	c := *s

	return &c
}

func (s *OrderStore) editable() error {
	p := s.Props()
	if p.Status != CreateOrder && p.Status != UpdateOrder {
		return errors.New("only draft stores can be edited")
	}

	return nil
}

func (s *OrderStore) commit(p OrderStoreProps, status Status, at time.Time) error {
	entry, err := NewOrderStatus(OrderStatusProps{
		Status:     status,
		OccurredAt: at,
	})
	if err != nil {
		return err
	}

	if len(p.History) > 0 && at.Before(p.History[len(p.History)-1].Props().OccurredAt) {
		return errors.New("status timestamps must be chronological")
	}

	p.Status = status
	p.History = append(append([]OrderStatus(nil), p.History...), entry)
	s.aggregate.Update(func(current *OrderStoreProps) { *current = p })

	return nil
}

func (s *OrderStore) AddOrder(props OrderProps) error {
	if err := s.editable(); err != nil {
		return err
	}

	p := s.Props()
	if !p.Order.ID().Empty() {
		return errors.New("order already added")
	}

	order, err := NewOrder(props)
	if err != nil {
		return err
	}

	p.Order = order

	return s.commit(p, UpdateOrder, time.Now().UTC())
}

func (s *OrderStore) AddVendor(props VendorProps) error {
	if err := s.editable(); err != nil {
		return err
	}

	p := s.Props()
	if !p.Vendor.ID().Empty() {
		return errors.New("store already has a vendor; create a separate OrderStore for another vendor")
	}

	if props.City != p.City {
		return errors.New("vendor and store cities must match")
	}

	vendor, err := NewVendor(props)
	if err != nil {
		return err
	}

	p.Vendor = vendor

	return s.commit(p, UpdateOrder, time.Now().UTC())
}

func (s *OrderStore) AddCustomer(props CustomerProps) error {
	if err := s.editable(); err != nil {
		return err
	}

	p := s.Props()
	if p.Order.ID().Empty() {
		return errors.New("add order before customer")
	}

	customer, err := NewCustomer(props)
	if err != nil {
		return err
	}

	p.Order.entity.Update(func(o *OrderProps) { o.Customer = customer })

	return s.commit(p, UpdateOrder, time.Now().UTC())
}

func (s *OrderStore) AddItems(vendorID uint, props []ItemProps) error {
	if err := s.editable(); err != nil {
		return err
	}

	p := s.Props()
	if p.Vendor.ID().Empty() || p.Vendor.Props().VendorId != vendorID {
		return errors.New("vendor does not belong to store")
	}

	if len(props) == 0 {
		return errors.New("items are required")
	}

	items := p.Vendor.Props().Items
	for _, itemProps := range props {
		item, err := NewItem(itemProps)
		if err != nil {
			return err
		}

		items = append(items, item)
	}

	p.Vendor.entity.Update(func(v *VendorProps) { v.Items = items })

	return s.commit(p, UpdateOrder, time.Now().UTC())
}

func (s *OrderStore) MarkReady(at time.Time) error {
	if err := s.editable(); err != nil {
		return err
	}

	p := s.Props()
	if err := p.Order.Validate(); err != nil {
		return err
	}

	if err := p.Order.Props().Customer.Validate(); err != nil {
		return err
	}

	if err := p.Vendor.Validate(); err != nil {
		return err
	}

	if len(p.Vendor.Props().Items) == 0 {
		return errors.New("store requires items")
	}

	return s.commit(p, OrderIsReady, at)
}

func (s *OrderStore) Assign(at time.Time) error {
	if s.Props().Status != OrderIsReady {
		return errors.New("only ready stores can be assigned")
	}

	return s.commit(s.Props(), Assigned, at)
}

func (s *OrderStore) PickUp(at time.Time) error {
	if s.Props().Status != Assigned {
		return errors.New("only assigned stores can be picked up")
	}

	return s.commit(s.Props(), PickedUp, at)
}

func (s *OrderStore) Complete(at time.Time) error {
	if s.Props().Status != PickedUp {
		return errors.New("only picked-up stores can be delivered")
	}

	return s.commit(s.Props(), Delivered, at)
}

func (s *OrderStore) Cancel(at time.Time) error {
	p := s.Props()
	if p.Status == Delivered || p.Status == Cancelled || p.Status == PickedUp {
		return errors.New("store cannot be cancelled in its current status")
	}

	return s.commit(p, Cancelled, at)
}
