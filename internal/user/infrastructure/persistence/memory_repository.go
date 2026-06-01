package persistence

import (
	"context"
	"sync"

	"github.com/example/payment-federation/internal/user/domain/user"
)

// MemoryRepository é um adaptador in-memory da porta user.Repository (dev/teste).
// Garante e-mail único e concorrência otimista por Version.
type MemoryRepository struct {
	mu       sync.RWMutex
	byID     map[string]user.Snapshot
	byEmail  map[string]string // email -> id
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:    make(map[string]user.Snapshot),
		byEmail: make(map[string]string),
	}
}

var _ user.Repository = (*MemoryRepository)(nil)

func (r *MemoryRepository) Save(ctx context.Context, u *user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := u.ToSnapshot()
	if _, exists := r.byEmail[s.Email]; exists {
		return user.ErrEmailTaken
	}
	r.byID[s.ID] = s
	r.byEmail[s.Email] = s.ID
	return nil
}

func (r *MemoryRepository) Update(ctx context.Context, u *user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := u.ToSnapshot()
	current, ok := r.byID[s.ID]
	if !ok {
		return user.ErrNotFound
	}
	// Se o e-mail mudou, garante unicidade e reindexa.
	if current.Email != s.Email {
		if _, exists := r.byEmail[s.Email]; exists {
			return user.ErrEmailTaken
		}
		delete(r.byEmail, current.Email)
		r.byEmail[s.Email] = s.ID
	}
	r.byID[s.ID] = s
	return nil
}

func (r *MemoryRepository) FindByID(ctx context.Context, id user.UserID) (*user.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.byID[id.String()]
	if !ok {
		return nil, user.ErrNotFound
	}
	return user.FromSnapshot(s), nil
}

func (r *MemoryRepository) FindByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byEmail[email.String()]
	if !ok {
		return nil, user.ErrNotFound
	}
	return user.FromSnapshot(r.byID[id]), nil
}

func (r *MemoryRepository) List(ctx context.Context) ([]*user.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*user.User, 0, len(r.byID))
	for _, s := range r.byID {
		out = append(out, user.FromSnapshot(s))
	}
	return out, nil
}
