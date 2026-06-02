package graph

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/example/payment-federation/internal/app/graph/model"
	"github.com/example/payment-federation/internal/payments/application/event"
	"github.com/example/payment-federation/internal/shared/messaging"
)

// PaymentHub é um broadcaster in-process para a subscription de pagamentos.
// Cada assinante filtra por um orderId (== chave de idempotência do pagamento)
// e tem o seu canal; envios são NÃO-BLOQUEANTES — um assinante lento nunca trava
// o pump do EventBus. Thread-safe.
type PaymentHub struct {
	mu     sync.Mutex
	nextID int
	subs   map[int]subscription
}

type subscription struct {
	orderID string
	ch      chan *model.Payment
}

func NewPaymentHub() *PaymentHub {
	return &PaymentHub{subs: make(map[int]subscription)}
}

// Subscribe registra um assinante para um orderId e devolve o canal de leitura.
// Ao cancelar o ctx (fim da subscription / cliente desconectou), o assinante é
// removido e o canal fechado — o que encerra a subscription do lado do gqlgen.
func (h *PaymentHub) Subscribe(ctx context.Context, orderID string) <-chan *model.Payment {
	ch := make(chan *model.Payment, 16) // buffer evita perder eventos entre Publish e leitura

	h.mu.Lock()
	id := h.nextID
	h.nextID++
	h.subs[id] = subscription{orderID: orderID, ch: ch}
	h.mu.Unlock()

	go func() {
		<-ctx.Done()
		h.mu.Lock()
		delete(h.subs, id)
		close(ch)
		h.mu.Unlock()
	}()
	return ch
}

// NumSubscribers devolve o número de assinantes ativos (observabilidade/testes).
func (h *PaymentHub) NumSubscribers() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}

// Publish entrega o pagamento aos assinantes cujo orderId casa com a chave
// (não-bloqueante).
func (h *PaymentHub) Publish(orderID string, p *model.Payment) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, s := range h.subs {
		if s.orderID != orderID {
			continue
		}
		select {
		case s.ch <- p:
		default: // assinante lento: descarta para ele (mantém o sistema responsivo)
		}
	}
}

// RunPaymentEventStream conecta um CANAL gerado pelo PRÓPRIO EventBus (Watermill
// message.Subscriber.Subscribe) ao PaymentHub: em vez de registrar um handler do
// cqrs + reconsultar o banco, "ouve" o evento PaymentIntegrationEvent como um
// <-chan *message.Message e, na captura (sucesso), entrega os dados do pagamento
// — que já vêm no próprio evento — aos assinantes da subscription.
//
// É o mesmo tópico que o consumidor payment_projection observa (fan-out do
// GoChannel: cada Subscribe recebe sua cópia). O Ack vem DEPOIS de entregar ao
// hub — coerente com o block-until-ack do publisher.
func RunPaymentEventStream(lc fx.Lifecycle, sub message.Subscriber, hub *PaymentHub, log *zap.Logger) error {
	l := log.Named("payment_event_stream")
	topic := messaging.Marshaler().Name(event.PaymentIntegrationEvent{})

	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			messages, err := sub.Subscribe(ctx, topic)
			if err != nil {
				cancel()
				return err
			}
			l.Info("ouvindo eventos de pagamento como canal", zap.String("topic", topic))
			go pumpPaymentEvents(messages, hub, l)
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})
	return nil
}

func pumpPaymentEvents(messages <-chan *message.Message, hub *PaymentHub, log *zap.Logger) {
	for msg := range messages {
		var evt event.PaymentIntegrationEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			log.Error("falha ao desserializar evento de pagamento", zap.Error(err))
			msg.Ack()
			continue
		}
		if evt.Name == "payment.captured" { // "processado com sucesso"
			hub.Publish(evt.IdempotencyKey, paymentFromEvent(&evt))
		}
		msg.Ack()
	}
}

// paymentFromEvent monta o model GraphQL a partir do SNAPSHOT do evento — sem
// reconsultar o banco.
func paymentFromEvent(evt *event.PaymentIntegrationEvent) *model.Payment {
	p := &model.Payment{
		ID:            evt.PaymentID,
		CustomerID:    evt.CustomerID,
		AmountCents:   evt.AmountCents,
		Currency:      evt.Currency,
		RefundedCents: evt.RefundedCents,
		Status:        evt.Status,
	}
	if evt.GatewayRef != "" {
		ref := evt.GatewayRef
		p.GatewayRef = &ref
	}
	return p
}
