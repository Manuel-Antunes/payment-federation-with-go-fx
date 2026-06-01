// Package bus registra os handlers de query do módulo de usuário no QueryBus
// genérico compartilhado (internal/shared/cqrs).
package bus

import (
	"context"

	"github.com/example/payment-federation/internal/shared/cqrs"
	"github.com/example/payment-federation/internal/user/application/query"
)

// RegisterQueries registra os handlers de leitura no QueryBus genérico.
func RegisterQueries(
	qb *cqrs.QueryBus,
	getByID *query.GetUserHandler,
	getByEmail *query.GetUserByEmailHandler,
	getByIDs *query.GetUsersByIDsHandler,
	list *query.ListUsersHandler,
	exists *query.ExistsHandler,
) error {
	if err := cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetUser) (query.UserView, error) {
		return getByID.Handle(ctx, q)
	}); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetUserByEmail) (query.UserView, error) {
		return getByEmail.Handle(ctx, q)
	}); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetUsersByIDs) ([]query.UserView, error) {
		return getByIDs.Handle(ctx, q)
	}); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, func(ctx context.Context, q query.ListUsers) ([]query.UserView, error) {
		return list.Handle(ctx, q)
	}); err != nil {
		return err
	}
	return cqrs.RegisterQuery(qb, func(ctx context.Context, q query.Exists) (bool, error) {
		return exists.Handle(ctx, q)
	})
}
