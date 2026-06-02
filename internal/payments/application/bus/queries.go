// Package bus liga os use-cases do módulo de pagamentos aos barramentos de
// CQRS (internal/shared/cqrs e o backbone Watermill do kernel). Concentra o
// REGISTRO de commands, queries e events — depende só do kernel e da própria
// camada de aplicação, nunca da infraestrutura do módulo.
package bus

import (
	"github.com/example/payment-federation/internal/payments/application/query"
	"github.com/example/payment-federation/internal/shared/cqrs"
)

// RegisterQueries registra os handlers de query (pagamento e pedido) no QueryBus.
// As queries devolvem os AGREGADOS de domínio; a tradução para a saída é na borda.
func RegisterQueries(
	qb *cqrs.QueryBus,
	getPayment *query.GetPaymentHandler,
	getPaymentByKey *query.GetPaymentByKeyHandler,
	getOrder *query.GetOrderHandler,
	getOrderByKey *query.GetOrderByKeyHandler,
	getOrdersByIDs *query.GetOrdersByIDsHandler,
) error {
	if err := cqrs.RegisterQuery(qb, getPayment); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, getPaymentByKey); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, getOrder); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, getOrdersByIDs); err != nil {
		return err
	}
	return cqrs.RegisterQuery(qb, getOrderByKey)
}
