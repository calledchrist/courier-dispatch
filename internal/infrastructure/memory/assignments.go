package memory

import (
	"context"
	"errors"
	"sort"

	domain "github.com/calledchrist/courier-dispatch/internal/app/dispatch/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/dispatch/ports"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

type Assignments struct {
	store *Store
}

func (s *Store) Assignments() *Assignments {
	return &Assignments{store: s}
}

var _ ports.Repository = (*Assignments)(nil)

func (r *Assignments) Get(ctx context.Context, id ddd.ID) (*domain.Assignment, error) {
	var result *domain.Assignment
	err := r.store.access(ctx, false, func(st *state) error {
		m, ok := st.assignments[id]
		if !ok {
			return ErrNotFound
		}

		result = m.Clone()

		return nil
	})

	return result, err
}

func (r *Assignments) Save(ctx context.Context, model *domain.Assignment) error {
	if model == nil || model.ID().Empty() {
		return errors.New("valid aggregate is required")
	}

	return r.store.access(ctx, true, func(st *state) error {
		for id, existing := range st.assignments {
			if id != model.ID() && existing.Props().DeliveryID == model.Props().DeliveryID {
				return errors.New("delivery already has an assignment")
			}
		}

		st.assignments[model.ID()] = model.Clone()

		return nil
	})
}

func (r *Assignments) List(ctx context.Context) ([]*domain.Assignment, error) {
	result := []*domain.Assignment{}
	err := r.store.access(ctx, false, func(st *state) error {
		for _, m := range st.assignments {
			result = append(result, m.Clone())
		}

		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].ID() < result[j].ID() })

	return result, err
}
