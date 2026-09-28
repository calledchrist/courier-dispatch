package memory

import (
	"context"
	"errors"
	"sort"

	domain "github.com/calledchrist/courier-dispatch/internal/app/order/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/order/ports"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

type Orders struct {
	store *Store
}

func (s *Store) Orders() *Orders {
	return &Orders{store: s}
}

var _ ports.Repository = (*Orders)(nil)

func (r *Orders) Get(ctx context.Context, id ddd.ID) (*domain.OrderStore, error) {
	var result *domain.OrderStore
	err := r.store.access(ctx, false, func(st *state) error {
		m, ok := st.orders[id]
		if !ok {
			return ErrNotFound
		}

		result = m.Clone()

		return nil
	})

	return result, err
}

func (r *Orders) Save(ctx context.Context, model *domain.OrderStore) error {
	if model == nil || model.ID().Empty() {
		return errors.New("valid aggregate is required")
	}

	return r.store.access(ctx, true, func(st *state) error {
		for id, existing := range st.orders {
			if id != model.ID() && existing.Props().OrderStoreId == model.Props().OrderStoreId {
				return errors.New("duplicate external order-store ID")
			}
		}

		st.orders[model.ID()] = model.Clone()

		return nil
	})
}

func (r *Orders) List(ctx context.Context) ([]*domain.OrderStore, error) {
	result := []*domain.OrderStore{}
	err := r.store.access(ctx, false, func(st *state) error {
		for _, m := range st.orders {
			result = append(result, m.Clone())
		}

		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].ID() < result[j].ID() })

	return result, err
}
