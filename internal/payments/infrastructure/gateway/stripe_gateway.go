package gateway

import (
	"context"
	"fmt"

	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// StripeGateway é um SEGUNDO provedor concreto, demonstrando que o mesmo
// contrato genérico (payment.Gateway) atende a qualquer integração. Aqui é um
// stub que simula a API da Stripe; numa implementação real, o client HTTP da
// Stripe entraria neste adaptador — e o domínio continuaria intocado.
type StripeGateway struct {
	apiKey string
	// declineOver simula recusa acima de um teto (regra do PROVEDOR, não do
	// domínio). 0 = nunca recusa.
	declineOverCents int64
}

var _ payment.Gateway = (*StripeGateway)(nil)

func NewStripeGateway(apiKey string, declineOverCents int64) *StripeGateway {
	return &StripeGateway{apiKey: apiKey, declineOverCents: declineOverCents}
}

func (g *StripeGateway) Name() string { return "stripe" }

func (g *StripeGateway) Authorize(ctx context.Context, in payment.AuthorizeInput) (payment.AuthorizeResult, error) {
	if g.declineOverCents > 0 && in.Amount.AmountCents() > g.declineOverCents {
		return payment.AuthorizeResult{Approved: false, DeclineReason: "limit_exceeded"}, nil
	}
	// Idempotency-Key header garante segurança em retries do lado da Stripe.
	ref := payment.GatewayReference(fmt.Sprintf("ch_%s", in.IdempotencyKey.String()))
	return payment.AuthorizeResult{Approved: true, GatewayRef: ref}, nil
}

func (g *StripeGateway) Capture(ctx context.Context, in payment.CaptureInput) error { return nil }
func (g *StripeGateway) Refund(ctx context.Context, in payment.RefundInput) error   { return nil }
