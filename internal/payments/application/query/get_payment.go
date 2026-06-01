package query

import (
	"context"

	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// PaymentView é o modelo de LEITURA (read model do CQRS). É um DTO plano,
// otimizado para consumo; nunca expomos o agregado diretamente.
type PaymentView struct {
	ID             string
	IdempotencyKey string
	CustomerID     string
	AmountCents    int64
	Currency       string
	RefundedCents  int64
	Status         string
	GatewayRef     string
}

func viewFrom(p *payment.Payment) PaymentView {
	return PaymentView{
		ID:             p.ID().String(),
		IdempotencyKey: p.IdempotencyKey().String(),
		CustomerID:     p.CustomerID(),
		AmountCents:    p.Amount().AmountCents(),
		Currency:       string(p.Amount().Currency()),
		RefundedCents:  p.Refunded().AmountCents(),
		Status:         string(p.Status()),
		GatewayRef:     p.GatewayRef().String(),
	}
}

// GetPayment é uma query simples por id.
type GetPayment struct {
	PaymentID string
}

type GetPaymentHandler struct {
	repo payment.Repository
}

func NewGetPaymentHandler(repo payment.Repository) *GetPaymentHandler {
	return &GetPaymentHandler{repo: repo}
}

func (h *GetPaymentHandler) Handle(ctx context.Context, q GetPayment) (PaymentView, error) {
	p, err := h.repo.FindByID(ctx, payment.PaymentID(q.PaymentID))
	if err != nil {
		return PaymentView{}, err
	}
	return viewFrom(p), nil
}

// GetPaymentByKey resolve um pagamento pela chave de idempotência. É o read
// model usado no read-after-write da mutation processPayment: como o COMMAND é
// despachado pelo CommandBus (Watermill) — e o id do pagamento é gerado DENTRO
// do handler —, a interface não conhece o id; relê pela chave que ela própria
// enviou (funciona tanto para criação quanto para replay idempotente).
type GetPaymentByKey struct {
	IdempotencyKey string
}

type GetPaymentByKeyHandler struct {
	repo payment.Repository
}

func NewGetPaymentByKeyHandler(repo payment.Repository) *GetPaymentByKeyHandler {
	return &GetPaymentByKeyHandler{repo: repo}
}

func (h *GetPaymentByKeyHandler) Handle(ctx context.Context, q GetPaymentByKey) (PaymentView, error) {
	key, err := payment.NewIdempotencyKey(q.IdempotencyKey)
	if err != nil {
		return PaymentView{}, err
	}
	p, err := h.repo.FindByIdempotencyKey(ctx, key)
	if err != nil {
		return PaymentView{}, err
	}
	return viewFrom(p), nil
}
