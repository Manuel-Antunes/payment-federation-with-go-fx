package command

import (
	"context"

	"github.com/example/payment-federation/internal/payments/application/port"
	"github.com/example/payment-federation/internal/shared/clock"
	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// RefundPayment é o command de reembolso (total ou parcial).
type RefundPayment struct {
	PaymentID   string
	AmountCents int64
}

type RefundPaymentResult struct {
	PaymentID     string
	Status        string
	RefundedCents int64
}

type RefundPaymentHandler struct {
	repo      payment.Repository
	gateways  payment.GatewayProvider
	clock     clock.Clock
	publisher port.EventPublisher
}

func NewRefundPaymentHandler(
	repo payment.Repository,
	gateways payment.GatewayProvider,
	clock clock.Clock,
	publisher port.EventPublisher,
) *RefundPaymentHandler {
	return &RefundPaymentHandler{repo: repo, gateways: gateways, clock: clock, publisher: publisher}
}

func (h *RefundPaymentHandler) Handle(ctx context.Context, cmd RefundPayment) (RefundPaymentResult, error) {
	p, err := h.repo.FindByID(ctx, payment.PaymentID(cmd.PaymentID))
	if err != nil {
		return RefundPaymentResult{}, err
	}

	amount, err := payment.NewMoney(cmd.AmountCents, p.Amount().Currency())
	if err != nil {
		return RefundPaymentResult{}, err
	}

	clk := payment.Clock(h.clock.Now)

	// REGRA DE NEGÓCIO PRIMEIRO: o agregado valida o limite de reembolso e a
	// transição de estado ANTES de qualquer chamada externa ou persistência.
	if err := p.Refund(amount, clk); err != nil {
		return RefundPaymentResult{}, err
	}

	// Efeito externo no gateway (idempotente pela GatewayRef).
	gw, err := h.gateways.Resolve(gatewayNameFor(p))
	if err != nil {
		gw = h.gateways.Active()
	}
	if err := gw.Refund(ctx, payment.RefundInput{GatewayRef: p.GatewayRef(), Amount: amount}); err != nil {
		return RefundPaymentResult{}, err
	}

	// Persistência por último, com concorrência otimista (Update usa Version).
	if err := h.repo.Update(ctx, p); err != nil {
		return RefundPaymentResult{}, err
	}

	if h.publisher != nil {
		_ = h.publisher.Publish(ctx, p.PullEvents()...)
	}

	return RefundPaymentResult{
		PaymentID:     p.ID().String(),
		Status:        string(p.Status()),
		RefundedCents: p.Refunded().AmountCents(),
	}, nil
}

// gatewayNameFor poderia derivar o provedor a partir de metadados do pagamento.
// Aqui usamos o ativo como padrão.
func gatewayNameFor(_ *payment.Payment) string { return "" }
