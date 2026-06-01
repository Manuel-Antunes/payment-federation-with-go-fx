package persistence

import (
	"context"
	"sync"

	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// MemoryRepository é um adaptador da porta payment.Repository para testes e
// desenvolvimento. Garante idempotência por IdempotencyKey e concorrência
// otimista por Version.
type MemoryRepository struct {
	mu      sync.RWMutex
	byID    map[string]payment.Snapshot
	byIdemp map[string]string // idempotencyKey -> id
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:    make(map[string]payment.Snapshot),
		byIdemp: make(map[string]string),
	}
}

// garante que satisfaz a porta em tempo de compilação.
var _ payment.Repository = (*MemoryRepository)(nil)

func (r *MemoryRepository) Save(ctx context.Context, p *payment.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := p.IdempotencyKey().String()
	if _, exists := r.byIdemp[key]; exists {
		return payment.ErrAlreadyExists
	}
	s := p.ToSnapshot()
	r.byID[s.ID] = s
	r.byIdemp[key] = s.ID
	return nil
}

func (r *MemoryRepository) Update(ctx context.Context, p *payment.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := p.ToSnapshot()
	current, ok := r.byID[s.ID]
	if !ok {
		return payment.ErrNotFound
	}
	// Concorrência otimista: a versão em memória + 1 deve bater com a nova.
	if current.Version+1 != s.Version {
		return payment.ErrInvalidTransition // versão divergente
	}
	r.byID[s.ID] = s
	return nil
}

func (r *MemoryRepository) FindByID(ctx context.Context, id payment.PaymentID) (*payment.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.byID[id.String()]
	if !ok {
		return nil, payment.ErrNotFound
	}
	return payment.FromSnapshot(s), nil
}

func (r *MemoryRepository) FindByIdempotencyKey(ctx context.Context, key payment.IdempotencyKey) (*payment.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byIdemp[key.String()]
	if !ok {
		return nil, payment.ErrNotFound
	}
	return payment.FromSnapshot(r.byID[id]), nil
}
