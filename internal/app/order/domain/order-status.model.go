package order_domain

import (
	"errors"
	"time"

	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

type Status string

const (
	CreateOrder  Status = "create_order"
	UpdateOrder  Status = "update_order"
	OrderIsReady Status = "order_is_ready"
	Assigned     Status = "assigned"
	PickedUp     Status = "picked_up"
	Delivered    Status = "delivered"
	Cancelled    Status = "cancelled"
)

type OrderStatusProps struct {
	Status     Status
	OccurredAt time.Time
}

// OrderStatus is an immutable entry in the store's history.
type OrderStatus struct {
	entity ddd.Entity[OrderStatusProps]
}

func NewOrderStatus(p OrderStatusProps) (OrderStatus, error) {
	e, err := ddd.NewEntity(ddd.NewID(), p, ValidateOrderStatus)
	if err != nil {
		return OrderStatus{}, err
	}

	return OrderStatus{entity: *e}, nil
}

func ValidateOrderStatus(p OrderStatusProps) error {
	if p.OccurredAt.IsZero() {
		return errors.New("status timestamp is required")
	}

	switch p.Status {
	case CreateOrder, UpdateOrder, OrderIsReady, Assigned, PickedUp, Delivered, Cancelled:
		return nil
	}

	return errors.New("invalid order status")
}

func (s OrderStatus) ID() ddd.ID {
	return s.entity.ID()
}

func (s OrderStatus) Props() OrderStatusProps {
	return s.entity.Props()
}
