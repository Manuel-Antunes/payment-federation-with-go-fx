package persistence

import (
	"context"
	"sync"

	"github.com/example/payment-federation/internal/payments/domain/order"
)

// OrderMemoryRepository é um adaptador in-memory da porta order.Repository.
// Garante idempotência por IdempotencyKey.
type OrderMemoryRepository struct {
	mu      sync.RWMutex
	byID    map[string]order.Snapshot
	byIdemp map[string]string // idempotencyKey -> id
}

func NewOrderMemoryRepository() *OrderMemoryRepository {
	return &OrderMemoryRepository{
		byID:    make(map[string]order.Snapshot),
		byIdemp: make(map[string]string),
	}
}

var _ order.Repository = (*OrderMemoryRepository)(nil)

func (r *OrderMemoryRepository) Save(ctx context.Context, o *order.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := o.IdempotencyKey().String()
	if _, exists := r.byIdemp[key]; exists {
		return order.ErrAlreadyExists
	}
	s := o.ToSnapshot()
	r.byID[s.ID] = s
	r.byIdemp[key] = s.ID
	return nil
}

func (r *OrderMemoryRepository) FindByID(ctx context.Context, id order.OrderID) (*order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.byID[id.String()]
	if !ok {
		return nil, order.ErrNotFound
	}
	return order.FromSnapshot(s), nil
}

func (r *OrderMemoryRepository) FindByIDs(ctx context.Context, ids []order.OrderID) ([]*order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*order.Order, 0, len(ids))
	for _, id := range ids {
		if s, ok := r.byID[id.String()]; ok {
			out = append(out, order.FromSnapshot(s))
		}
	}
	return out, nil
}

func (r *OrderMemoryRepository) FindByIdempotencyKey(ctx context.Context, key string) (*order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byIdemp[key]
	if !ok {
		return nil, order.ErrNotFound
	}
	return order.FromSnapshot(r.byID[id]), nil
}
