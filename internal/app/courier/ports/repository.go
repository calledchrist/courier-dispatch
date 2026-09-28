package courier_ports

import (
	"context"

	domain "github.com/calledchrist/courier-dispatch/internal/app/courier/domain"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

type Repository interface {
	Get(context.Context, ddd.ID) (*domain.Courier, error)
	Save(context.Context, *domain.Courier) error
	ListByCity(context.Context, string) ([]*domain.Courier, error)
}
