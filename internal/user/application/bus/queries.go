// Package bus registra os handlers de query do módulo de usuário no QueryBus
// genérico compartilhado (internal/shared/cqrs).
package bus

import (
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
	if err := cqrs.RegisterQuery(qb, getByID); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, getByEmail); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, getByIDs); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, list); err != nil {
		return err
	}
	return cqrs.RegisterQuery(qb, exists)
}
