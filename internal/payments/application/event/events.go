// Package event define os EVENTOS DE INTEGRAÇÃO publicados pelo módulo de
// pagamentos no barramento — o contrato estável e serializável que cruza a
// fronteira do domínio. As funções de mapeamento traduzem os eventos de domínio
// (que carregam Value Objects e campos não exportados) para estes DTOs planos.
package event

import (
	"time"

	"github.com/example/payment-federation/internal/payments/domain/order"
	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// PaymentIntegrationEvent é o evento de integração de pagamento.
type PaymentIntegrationEvent struct {
	Name       string    `json:"name"`       // ex.: "payment.captured"
	PaymentID  string    `json:"paymentId"`  // id do agregado afetado
	OccurredAt time.Time `json:"occurredAt"` // instante (UTC) do evento no domínio
}

// PaymentIntegrationFrom traduz um evento de domínio de pagamento para o DTO.
func PaymentIntegrationFrom(e payment.DomainEvent) PaymentIntegrationEvent {
	return PaymentIntegrationEvent{
		Name:       e.EventName(),
		PaymentID:  e.AggregateID().String(),
		OccurredAt: e.OccurredAt().UTC(),
	}
}

// OrderIntegrationEvent é o evento de integração de pedido. Carrega o que a
// saga de pagamento precisa para iniciar o pagamento do pedido.
type OrderIntegrationEvent struct {
	Name        string    `json:"name"`        // ex.: "order.created"
	OrderID     string    `json:"orderId"`     // id do pedido (usado como chave de idempotência do pagamento)
	CustomerID  string    `json:"customerId"`  // cliente do pedido
	AmountCents int64     `json:"amountCents"` // valor a cobrar
	Currency    string    `json:"currency"`    // moeda
	OccurredAt  time.Time `json:"occurredAt"`  // instante (UTC)
}

// OrderIntegrationFrom traduz um OrderCreated de domínio para o DTO.
func OrderIntegrationFrom(e order.OrderCreated) OrderIntegrationEvent {
	return OrderIntegrationEvent{
		Name:        e.EventName(),
		OrderID:     e.AggregateID().String(),
		CustomerID:  e.CustomerID,
		AmountCents: e.AmountCents,
		Currency:    e.Currency,
		OccurredAt:  e.OccurredAt().UTC(),
	}
}
