package memory

import (
	"context"
	"errors"
	"sort"

	domain "github.com/calledchrist/courier-dispatch/internal/app/delivery/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/delivery/ports"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

type Deliveries struct {
	store *Store
}

func (s *Store) Deliveries() *Deliveries {
	return &Deliveries{store: s}
}

var _ ports.Repository = (*Deliveries)(nil)

func (r *Deliveries) Get(ctx context.Context, id ddd.ID) (*domain.Delivery, error) {
	var result *domain.Delivery
	err := r.store.access(ctx, false, func(st *state) error {
		m, ok := st.deliveries[id]
		if !ok {
			return ErrNotFound
		}

		result = m.Clone()

		return nil
	})

	return result, err
}

func (r *Deliveries) Save(ctx context.Context, model *domain.Delivery) error {
	if model == nil || model.ID().Empty() {
		return errors.New("valid aggregate is required")
	}

	return r.store.access(ctx, true, func(st *state) error {
		for id, existing := range st.deliveries {
			if id != model.ID() && existing.Active() && model.Active() && existing.Props().OrderStoreID == model.Props().OrderStoreID {
				return errors.New("store already has an active delivery")
			}
		}

		st.deliveries[model.ID()] = model.Clone()

		return nil
	})
}

func (r *Deliveries) active(ctx context.Context, match func(domain.DeliveryProps) bool) ([]*domain.Delivery, error) {
	result := []*domain.Delivery{}
	err := r.store.access(ctx, false, func(st *state) error {
		for _, m := range st.deliveries {
			if m.Active() && match(m.Props()) {
				result = append(result, m.Clone())
			}
		}

		return nil
	})
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i].Props(), result[j].Props()
		if a.AssignedAt.Equal(b.AssignedAt) {
			return result[i].ID() < result[j].ID()
		}

		return a.AssignedAt.Before(b.AssignedAt)
	})

	return result, err
}

func (r *Deliveries) ActiveByCourier(ctx context.Context, id ddd.ID) ([]*domain.Delivery, error) {
	return r.active(ctx, func(p domain.DeliveryProps) bool { return p.CourierID == id })
}

func (r *Deliveries) ActiveByStore(ctx context.Context, id ddd.ID) ([]*domain.Delivery, error) {
	return r.active(ctx, func(p domain.DeliveryProps) bool { return p.OrderStoreID == id })
}
