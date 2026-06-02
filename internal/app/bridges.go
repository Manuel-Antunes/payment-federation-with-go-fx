package app

import (
	"context"

	paymentsport "github.com/example/payment-federation/internal/payments/application/port"
	"github.com/example/payment-federation/internal/shared/cqrs"
	uquery "github.com/example/payment-federation/internal/user/application/query"
)

// customerDirectory é o BRIDGE cross-module que implementa a porta
// payments/application.CustomerDirectory consultando o módulo de usuário via o
// QueryBus genérico compartilhado. Vive na composição (não acopla os módulos:
// pagamentos depende só da sua porta; usuário não conhece pagamentos).
type customerDirectory struct {
	queries *cqrs.QueryBus
}

var _ paymentsport.CustomerDirectory = (*customerDirectory)(nil)

func newCustomerDirectory(queries *cqrs.QueryBus) *customerDirectory {
	return &customerDirectory{queries: queries}
}

func (d *customerDirectory) Exists(ctx context.Context, customerID string) (bool, error) {
	return cqrs.ExecuteQuery[uquery.Exists, bool](ctx, d.queries, uquery.Exists{UserID: customerID})
}
