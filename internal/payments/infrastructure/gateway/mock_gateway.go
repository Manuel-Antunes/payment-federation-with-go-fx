package gateway

import (
	"context"
	"sync"

	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// MockGateway é uma implementação concreta do serviço genérico payment.Gateway.
// Aprova tudo e é idempotente por IdempotencyKey (mesma chave => mesma ref).
// Serve como um dos provedores intercambiáveis em runtime.
type MockGateway struct {
	mu   sync.Mutex
	refs map[string]payment.GatewayReference
}

var _ payment.Gateway = (*MockGateway)(nil)

func NewMockGateway() *MockGateway {
	return &MockGateway{refs: make(map[string]payment.GatewayReference)}
}

func (g *MockGateway) Name() string { return "mock" }

func (g *MockGateway) Authorize(ctx context.Context, in payment.AuthorizeInput) (payment.AuthorizeResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := in.IdempotencyKey.String()
	ref, ok := g.refs[key]
	if !ok {
		ref = payment.GatewayReference("mock_" + in.PaymentID.String())
		g.refs[key] = ref
	}
	return payment.AuthorizeResult{Approved: true, GatewayRef: ref}, nil
}

func (g *MockGateway) Capture(ctx context.Context, in payment.CaptureInput) error { return nil }
func (g *MockGateway) Refund(ctx context.Context, in payment.RefundInput) error   { return nil }
