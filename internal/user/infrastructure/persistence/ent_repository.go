// Package persistence implementa as portas de repositório do módulo de usuário
// sobre o ent (Postgres). O mapeamento ent <-> domínio fica em user_schema.go.
package persistence

import (
	"context"

	entx "github.com/example/payment-federation/internal/user/infrastructure/ent"
	entuser "github.com/example/payment-federation/internal/user/infrastructure/ent/user"
	"github.com/example/payment-federation/internal/user/domain/user"
)

// EntRepository é o adaptador ent da porta user.Repository.
type EntRepository struct {
	client *entx.Client
	mapper userSchema
}

func NewEntRepository(client *entx.Client) *EntRepository {
	return &EntRepository{client: client}
}

var _ user.Repository = (*EntRepository)(nil)

func (r *EntRepository) Save(ctx context.Context, u *user.User) error {
	_, err := r.mapper.toCreate(r.client, u).Save(ctx)
	if entx.IsConstraintError(err) {
		return user.ErrEmailTaken // e-mail único violado
	}
	return err
}

func (r *EntRepository) Update(ctx context.Context, u *user.User) error {
	s := u.ToSnapshot()
	err := r.client.User.UpdateOneID(s.ID).
		SetName(s.Name).
		SetEmail(s.Email).
		SetUpdatedAt(s.UpdatedAt).
		SetVersion(s.Version).
		Exec(ctx)
	switch {
	case entx.IsNotFound(err):
		return user.ErrNotFound
	case entx.IsConstraintError(err):
		return user.ErrEmailTaken
	default:
		return err
	}
}

func (r *EntRepository) FindByID(ctx context.Context, id user.UserID) (*user.User, error) {
	row, err := r.client.User.Get(ctx, id.String())
	if entx.IsNotFound(err) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.mapper.toDomain(row), nil
}

func (r *EntRepository) FindByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	row, err := r.client.User.Query().Where(entuser.Email(email.String())).Only(ctx)
	if entx.IsNotFound(err) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.mapper.toDomain(row), nil
}

func (r *EntRepository) FindByIDs(ctx context.Context, ids []user.UserID) ([]*user.User, error) {
	raw := make([]string, 0, len(ids))
	for _, id := range ids {
		raw = append(raw, id.String())
	}
	rows, err := r.client.User.Query().Where(entuser.IDIn(raw...)).All(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomainSlice(rows), nil
}

func (r *EntRepository) List(ctx context.Context) ([]*user.User, error) {
	rows, err := r.client.User.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	return r.toDomainSlice(rows), nil
}

func (r *EntRepository) toDomainSlice(rows []*entx.User) []*user.User {
	out := make([]*user.User, 0, len(rows))
	for _, row := range rows {
		out = append(out, r.mapper.toDomain(row))
	}
	return out
}
