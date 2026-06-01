// Package bus liga os use-cases do módulo de pagamentos aos barramentos de
// CQRS (internal/shared/cqrs e o backbone Watermill do kernel). Concentra o
// REGISTRO de commands, queries e events — depende só do kernel e da própria
// camada de aplicação, nunca da infraestrutura do módulo.
package bus

import (
	"context"

	"github.com/example/payment-federation/internal/payments/application/query"
	"github.com/example/payment-federation/internal/payments/domain/order"
	"github.com/example/payment-federation/internal/payments/domain/payment"
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
	if err := cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetPayment) (*payment.Payment, error) {
		return getPayment.Handle(ctx, q)
	}); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetPaymentByKey) (*payment.Payment, error) {
		return getPaymentByKey.Handle(ctx, q)
	}); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetOrder) (*order.Order, error) {
		return getOrder.Handle(ctx, q)
	}); err != nil {
		return err
	}
	if err := cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetOrdersByIDs) ([]*order.Order, error) {
		return getOrdersByIDs.Handle(ctx, q)
	}); err != nil {
		return err
	}
	return cqrs.RegisterQuery(qb, func(ctx context.Context, q query.GetOrderByKey) (*order.Order, error) {
		return getOrderByKey.Handle(ctx, q)
	})
}
