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
	if err := cqrs.RegisterEvent(events, "payment_projection",
		paymentProjection{log: log.Named("payment_projection")}); err != nil {
		return err
	}

	return cqrs.RegisterEvent(events, "process_payment_on_order_created",
		orderPaymentSaga{commands: commands, log: log.Named("order_payment_saga")})
}

// paymentProjection consome eventos de integração de pagamento (observabilidade
// / read models).
type paymentProjection struct {
	log *zap.Logger
}

func (p paymentProjection) Handle(ctx context.Context, evt *event.PaymentIntegrationEvent) error {
	p.log.Info("payment_event_consumed",
		zap.String("event", evt.Name),
		zap.String("payment_id", evt.PaymentID),
		zap.Time("occurred_at", evt.OccurredAt),
	)
	return nil
}

// orderPaymentSaga é a SAGA pedido->pagamento: ao receber OrderCreated, dispara
// o ProcessPayment usando o id do pedido como chave de idempotência (estável =>
// retries são idempotentes).
type orderPaymentSaga struct {
	commands *cqrs.CommandBus
	log      *zap.Logger
}

func (s orderPaymentSaga) Handle(ctx context.Context, evt *event.OrderIntegrationEvent) error {
	s.log.Info("order_created_triggering_payment",
		zap.String("order_id", evt.OrderID),
		zap.String("idempotence_key", evt.IdempotenceKey),
		zap.String("customer_id", evt.CustomerID),
	)
	// A chave de idempotência do pagamento é a chave EFETIVA do pedido (a custom
	// fornecida pelo cliente ou, na ausência, o id do pedido).
	return s.commands.Dispatch(ctx, command.ProcessPayment{
		IdempotencyKey: evt.IdempotenceKey,
		CustomerID:     evt.CustomerID,
		AmountCents:    evt.AmountCents,
		Currency:       evt.Currency,
	})
}
