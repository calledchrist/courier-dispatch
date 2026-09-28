package memory

import (
	"context"
	"errors"
	"sort"

	domain "github.com/calledchrist/courier-dispatch/internal/app/courier/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/courier/ports"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

type Couriers struct {
	store *Store
}

func (s *Store) Couriers() *Couriers {
	return &Couriers{store: s}
}

var _ ports.Repository = (*Couriers)(nil)

func (r *Couriers) Get(ctx context.Context, id ddd.ID) (*domain.Courier, error) {
	var result *domain.Courier
	err := r.store.access(ctx, false, func(st *state) error {
		m, ok := st.couriers[id]
		if !ok {
			return ErrNotFound
		}

		result = m.Clone()

		return nil
	})

	return result, err
}

func (r *Couriers) Save(ctx context.Context, model *domain.Courier) error {
	if model == nil || model.ID().Empty() {
		return errors.New("valid aggregate is required")
	}

	return r.store.access(ctx, true, func(st *state) error {
		st.couriers[model.ID()] = model.Clone()

		return nil
	})
}

func (r *Couriers) ListByCity(ctx context.Context, city string) ([]*domain.Courier, error) {
	result := []*domain.Courier{}
	err := r.store.access(ctx, false, func(st *state) error {
		for _, m := range st.couriers {
			if m.Props().City == city {
				result = append(result, m.Clone())
			}
		}

		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].ID() < result[j].ID() })

	return result, err
}
