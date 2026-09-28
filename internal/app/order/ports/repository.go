package order_ports

import (
	"context"

	domain "github.com/calledchrist/courier-dispatch/internal/app/order/domain"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

type Repository interface {
	Get(context.Context, ddd.ID) (*domain.OrderStore, error)
	Save(context.Context, *domain.OrderStore) error
	List(context.Context) ([]*domain.OrderStore, error)
}
