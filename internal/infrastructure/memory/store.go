// Package memory implements application ports with isolated in-memory transactions.
package memory

import (
	"context"
	"errors"
	"sync"

	courier "github.com/calledchrist/courier-dispatch/internal/app/courier/domain"
	delivery "github.com/calledchrist/courier-dispatch/internal/app/delivery/domain"
	dispatch "github.com/calledchrist/courier-dispatch/internal/app/dispatch/domain"
	order "github.com/calledchrist/courier-dispatch/internal/app/order/domain"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
	"github.com/calledchrist/courier-dispatch/internal/shared/uow"
)

var ErrNotFound = errors.New("aggregate not found")

var ErrTransactionRequired = errors.New("writes require an active unit of work")

type state struct {
	orders      map[ddd.ID]*order.OrderStore
	couriers    map[ddd.ID]*courier.Courier
	deliveries  map[ddd.ID]*delivery.Delivery
	assignments map[ddd.ID]*dispatch.Assignment
}

func newState() *state {
	return &state{
		orders:      map[ddd.ID]*order.OrderStore{},
		couriers:    map[ddd.ID]*courier.Courier{},
		deliveries:  map[ddd.ID]*delivery.Delivery{},
		assignments: map[ddd.ID]*dispatch.Assignment{},
	}
}

func (s *state) clone() *state {
	c := newState()
	for id, v := range s.orders {
		c.orders[id] = v.Clone()
	}

	for id, v := range s.couriers {
		c.couriers[id] = v.Clone()
	}

	for id, v := range s.deliveries {
		c.deliveries[id] = v.Clone()
	}

	for id, v := range s.assignments {
		c.assignments[id] = v.Clone()
	}

	return c
}

type Store struct {
	mu   sync.Mutex
	data *state
}

func New() *Store {
	return &Store{data: newState()}
}

var _ uow.UnitOfWork = (*Store)(nil)

type transactionKey struct{}

type transaction struct {
	owner  *Store
	data   *state
	closed bool
}

// Do serializes transactions, copies state, and publishes only a successful callback.
// Repositories must use the supplied context. The callback is synchronous;
// do not retain the context or share it with goroutines.
func (s *Store) Do(ctx context.Context, work func(context.Context) error) error {
	if ctx.Value(transactionKey{}) != nil {
		return errors.New("nested units of work are unsupported")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}

	tx := &transaction{
		owner: s,
		data:  s.data.clone(),
	}
	defer func() { tx.closed = true }()
	if err := work(context.WithValue(ctx, transactionKey{}, tx)); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	s.data = tx.data.clone()

	return nil
}

func (s *Store) access(ctx context.Context, write bool, fn func(*state) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if tx, ok := ctx.Value(transactionKey{}).(*transaction); ok {
		if tx.owner != s || tx.closed {
			return ErrTransactionRequired
		}

		return fn(tx.data)
	}

	if write {
		return ErrTransactionRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}

	return fn(s.data)
}
