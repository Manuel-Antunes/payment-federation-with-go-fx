package query

import (
	"context"

	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// GetPayment é uma query simples por id. Devolve o AGREGADO de domínio; a
// tradução para o tipo de saída (GraphQL) acontece na borda (interface).
type GetPayment struct {
	PaymentID string
}

type GetPaymentHandler struct {
	repo payment.Repository
}

func NewGetPaymentHandler(repo payment.Repository) *GetPaymentHandler {
	return &GetPaymentHandler{repo: repo}
}

func (h *GetPaymentHandler) Handle(ctx context.Context, q GetPayment) (*payment.Payment, error) {
	return h.repo.FindByID(ctx, payment.PaymentID(q.PaymentID))
}

// GetPaymentByKey resolve um pagamento pela chave de idempotência. É o
// read-after-write da mutation processPayment: como o COMMAND é despachado pelo
// CommandBus (Watermill) — e o id do pagamento é gerado DENTRO do handler —, a
// interface não conhece o id; relê pela chave que ela própria enviou (funciona
// tanto para criação quanto para replay idempotente).
type GetPaymentByKey struct {
	IdempotencyKey string
}

type GetPaymentByKeyHandler struct {
	repo payment.Repository
}

func NewGetPaymentByKeyHandler(repo payment.Repository) *GetPaymentByKeyHandler {
	return &GetPaymentByKeyHandler{repo: repo}
}

func (h *GetPaymentByKeyHandler) Handle(ctx context.Context, q GetPaymentByKey) (*payment.Payment, error) {
	key, err := payment.NewIdempotencyKey(q.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	return h.repo.FindByIdempotencyKey(ctx, key)
}
