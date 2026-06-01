// Package bus liga os use-cases do módulo de pagamentos aos barramentos de
// CQRS (internal/shared/cqrs e o backbone Watermill do kernel). Concentra o
// REGISTRO de commands, queries e events — depende só do kernel e da própria
// camada de aplicação, nunca da infraestrutura do módulo.
package bus

import (
	"context"

	"github.com/example/payment-federation/internal/payments/application/query"
	"github.com/example/payment-federation/internal/shared/cqrs"
)

// RegisterQueries registra os handlers de query (pagamento e pedido) no QueryBus.
func RegisterQueries(
	qb *cqrs.QueryBus,
	getPayment *query.GetPaymentHandler,
	getPaymentByKey *query.GetPaymentByKeyHandler,
	getOrder *query.GetOrderHandler,
	getOrderByKey *query.GetOrderByKeyHandler,
	getOrdersByIDs *query.GetOrdersByIDsHandler,
) error {
	if err := cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetPayment) (query.PaymentView, error) {
		return getPayment.Handle(ctx, q)
	}); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetPaymentByKey) (query.PaymentView, error) {
		return getPaymentByKey.Handle(ctx, q)
	}); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetOrder) (query.OrderView, error) {
		return getOrder.Handle(ctx, q)
	}); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetOrdersByIDs) ([]query.OrderView, error) {
		return getOrdersByIDs.Handle(ctx, q)
	}); err != nil {
		return err
	}
	return cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetOrderByKey) (query.OrderView, error) {
		return getOrderByKey.Handle(ctx, q)
	})
}
