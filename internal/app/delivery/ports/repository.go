package delivery_ports

import (
	"context"

	domain "github.com/calledchrist/courier-dispatch/internal/app/delivery/domain"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

type Repository interface {
	Get(context.Context, ddd.ID) (*domain.Delivery, error)
	Save(context.Context, *domain.Delivery) error
	ActiveByCourier(context.Context, ddd.ID) ([]*domain.Delivery, error)
	ActiveByStore(context.Context, ddd.ID) ([]*domain.Delivery, error)
}
