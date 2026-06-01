package bus

import (
	"context"

	"go.uber.org/zap"

	"github.com/example/payment-federation/internal/payments/application/command"
	"github.com/example/payment-federation/internal/payments/application/event"
	"github.com/example/payment-federation/internal/shared/cqrs"
)

// RegisterEvents registra os consumidores de eventos via cqrs.RegisterEvent
// (que encapsula o EventProcessor do Watermill):
//   - a projeção de pagamento (observabilidade / read models);
//   - a SAGA: ao receber OrderCreated, dispara o processamento do pagamento.
func RegisterEvents(events *cqrs.EventBus, commands *cqrs.CommandBus, log *zap.Logger) error {
	projection := log.Named("payment_projection")
	saga := log.Named("order_payment_saga")

	if err := cqrs.RegisterEvent(events, "payment_projection",
		func(ctx context.Context, evt *event.PaymentIntegrationEvent) error {
			projection.Info("payment_event_consumed",
				zap.String("event", evt.Name),
				zap.String("payment_id", evt.PaymentID),
				zap.Time("occurred_at", evt.OccurredAt),
			)
			return nil
		}); err != nil {
		return err
	}

	// SAGA pedido->pagamento: usa o id do pedido como chave de idempotência do
	// pagamento (estável => retries são idempotentes).
	return cqrs.RegisterEvent(events, "process_payment_on_order_created",
		func(ctx context.Context, evt *event.OrderIntegrationEvent) error {
			saga.Info("order_created_triggering_payment",
				zap.String("order_id", evt.OrderID),
				zap.String("customer_id", evt.CustomerID),
			)
			return commands.Dispatch(ctx, command.ProcessPayment{
				IdempotencyKey: evt.OrderID,
				CustomerID:     evt.CustomerID,
				AmountCents:    evt.AmountCents,
				Currency:       evt.Currency,
			})
		})
}
