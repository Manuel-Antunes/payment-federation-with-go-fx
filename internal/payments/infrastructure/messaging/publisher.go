// Package messaging contém os ADAPTADORES de mensageria do módulo de
// pagamentos: os publishers que implementam as portas de EventPublisher,
// mapeando eventos de domínio para os eventos de integração (application/event)
// e os despachando no EventBus do Watermill.
package messaging

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"go.uber.org/zap"

	"github.com/example/payment-federation/internal/payments/application/port"
	"github.com/example/payment-federation/internal/payments/application/event"
	"github.com/example/payment-federation/internal/payments/domain/order"
	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// WatermillPublisher implementa port.EventPublisher: publica os eventos
// de domínio de PAGAMENTO no EventBus do Watermill após a persistência.
type WatermillPublisher struct {
	bus *cqrs.EventBus
	log *zap.Logger
}

var _ port.EventPublisher = (*WatermillPublisher)(nil)

func NewWatermillPublisher(bus *cqrs.EventBus, log *zap.Logger) *WatermillPublisher {
	return &WatermillPublisher{bus: bus, log: log.Named("event_publisher")}
}

func (p *WatermillPublisher) Publish(ctx context.Context, pay *payment.Payment, events ...payment.DomainEvent) error {
	for _, e := range events {
		evt := event.PaymentIntegrationFrom(pay, e)
		if err := p.bus.Publish(ctx, evt); err != nil {
			p.log.Error("failed to publish payment event",
				zap.String("event", evt.Name), zap.String("payment_id", evt.PaymentID), zap.Error(err))
			return err
		}
	}
	return nil
}

// WatermillOrderPublisher implementa port.OrderEventPublisher: publica os
// eventos de domínio de PEDIDO no EventBus.
type WatermillOrderPublisher struct {
	bus *cqrs.EventBus
	log *zap.Logger
}

var _ port.OrderEventPublisher = (*WatermillOrderPublisher)(nil)

func NewWatermillOrderPublisher(bus *cqrs.EventBus, log *zap.Logger) *WatermillOrderPublisher {
	return &WatermillOrderPublisher{bus: bus, log: log.Named("order_event_publisher")}
}

func (p *WatermillOrderPublisher) Publish(ctx context.Context, events ...order.DomainEvent) error {
	for _, e := range events {
		oc, ok := e.(order.OrderCreated)
		if !ok {
			continue // só OrderCreated é publicado como evento de integração (por enquanto)
		}
		evt := event.OrderIntegrationFrom(oc)
		if err := p.bus.Publish(ctx, evt); err != nil {
			p.log.Error("failed to publish order event",
				zap.String("event", evt.Name), zap.String("order_id", evt.OrderID), zap.Error(err))
			return err
		}
	}
	return nil
}
