// Package event define os EVENTOS DE INTEGRAÇÃO publicados pelo módulo de
// pagamentos no barramento — o contrato estável e serializável que cruza a
// fronteira do domínio. As funções de mapeamento traduzem os eventos de domínio
// (que carregam Value Objects e campos não exportados) para estes DTOs planos.
package event

import (
	"time"

	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// PaymentIntegrationEvent é o evento de integração de pagamento. Além do nome do
// evento, carrega o SNAPSHOT do pagamento — para que consumidores (ex.: a
// subscription GraphQL) possam reagir com os dados completos SEM reconsultar o
// banco.
type PaymentIntegrationEvent struct {
	Name           string    `json:"name"`           // ex.: "payment.captured"
	PaymentID      string    `json:"paymentId"`      // id do agregado afetado
	IdempotencyKey string    `json:"idempotencyKey"` // == orderId quando criado pela saga
	CustomerID     string    `json:"customerId"`
	AmountCents    int64     `json:"amountCents"`
	Currency       string    `json:"currency"`
	RefundedCents  int64     `json:"refundedCents"`
	Status         string    `json:"status"`
	GatewayRef     string    `json:"gatewayRef,omitempty"`
	OccurredAt     time.Time `json:"occurredAt"` // instante (UTC) do evento no domínio
}

// PaymentIntegrationFrom traduz um evento de domínio para o DTO, anexando o
// snapshot do agregado (estado já persistido no momento da publicação).
func PaymentIntegrationFrom(p *payment.Payment, e payment.DomainEvent) PaymentIntegrationEvent {
	return PaymentIntegrationEvent{
		Name:           e.EventName(),
		PaymentID:      p.ID().String(),
		IdempotencyKey: p.IdempotencyKey().String(),
		CustomerID:     p.CustomerID(),
		AmountCents:    p.Amount().AmountCents(),
		Currency:       string(p.Amount().Currency()),
		RefundedCents:  p.Refunded().AmountCents(),
		Status:         string(p.Status()),
		GatewayRef:     p.GatewayRef().String(),
		OccurredAt:     e.OccurredAt().UTC(),
	}
}
