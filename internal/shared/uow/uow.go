package uow

import "context"

type UnitOfWork interface {
	Do(ctx context.Context, work func(context.Context) error) error
}
