package persistence

import (
	"github.com/example/payment-federation/internal/payments/domain/payment"
	entx "github.com/example/payment-federation/internal/payments/infrastructure/ent"
)

// paymentRow é alias da linha ent (evita colisão de nome de campo embutido com
// o agregado de domínio, ambos "Payment").
type paymentRow = entx.Payment

// paymentSchema casa a linha ent com o agregado de domínio, EMBUTINDO ambos, e
// concentra a conversão ent <-> domínio de pagamento.
type paymentSchema struct {
	*paymentRow
	*payment.Payment
}

func (paymentSchema) toDomain(r *entx.Payment) *payment.Payment {
	return payment.FromSnapshot(payment.Snapshot{
		ID:             r.ID,
		IdempotencyKey: r.IdempotencyKey,
		CustomerID:     r.CustomerID,
		AmountCents:    r.AmountCents,
		Currency:       r.Currency,
		RefundedCents:  r.RefundedCents,
		Status:         r.Status,
		GatewayRef:     r.GatewayRef,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
		Version:        r.Version,
	})
}

func (paymentSchema) toCreate(client *entx.Client, p *payment.Payment) *entx.PaymentCreate {
	s := p.ToSnapshot()
	return client.Payment.Create().
		SetID(s.ID).
		SetIdempotencyKey(s.IdempotencyKey).
		SetCustomerID(s.CustomerID).
		SetAmountCents(s.AmountCents).
		SetCurrency(s.Currency).
		SetRefundedCents(s.RefundedCents).
		SetStatus(s.Status).
		SetGatewayRef(s.GatewayRef).
		SetCreatedAt(s.CreatedAt).
		SetUpdatedAt(s.UpdatedAt).
		SetVersion(s.Version)
}
