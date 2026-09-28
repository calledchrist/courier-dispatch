package dispatch_ports

import (
	"context"

	domain "github.com/calledchrist/courier-dispatch/internal/app/dispatch/domain"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

type Repository interface {
	Get(context.Context, ddd.ID) (*domain.Assignment, error)
	Save(context.Context, *domain.Assignment) error
	List(context.Context) ([]*domain.Assignment, error)
}
