package payment

import "context"

// ---------------------------------------------------------------------------
// PORTA: Gateway genérico de processamento de pagamento.
//
// É deliberadamente genérica e agnóstica de provedor (Stripe, Adyen, Pix,
// mock...). O domínio só conhece este contrato. A escolha/implementação
// concreta é resolvida em RUNTIME pelo GatewayProvider (ver infra/registry),
// permitindo trocar o provedor sem recompilar nem reiniciar.
// ---------------------------------------------------------------------------

// AuthorizeInput é o que o domínio entrega ao gateway para autorizar.
type AuthorizeInput struct {
	PaymentID      PaymentID
	IdempotencyKey IdempotencyKey
	CustomerID     string
	Amount         Money
}

// AuthorizeResult é o que o gateway devolve.
type AuthorizeResult struct {
	Approved   bool
	GatewayRef GatewayReference
	// DeclineReason é preenchido quando Approved == false.
	DeclineReason string
}

type CaptureInput struct {
	GatewayRef GatewayReference
	Amount     Money
}

type RefundInput struct {
	GatewayRef GatewayReference
	Amount     Money
}

// Gateway é o serviço genérico substituível. Toda implementação deve ser
// idempotente em relação à IdempotencyKey/GatewayRef.
type Gateway interface {
	// Name identifica o provedor concreto (ex.: "stripe", "mock").
	Name() string

	Authorize(ctx context.Context, in AuthorizeInput) (AuthorizeResult, error)
	Capture(ctx context.Context, in CaptureInput) error
	Refund(ctx context.Context, in RefundInput) error
}

// GatewayProvider resolve, em runtime, qual Gateway usar. A aplicação depende
// desta porta — não de um Gateway fixo — para poder trocar o provedor ativo
// dinamicamente (feature flag, header da requisição, rollout etc.).
type GatewayProvider interface {
	// Active devolve o gateway atualmente selecionado.
	Active() Gateway
	// Resolve devolve um gateway específico por nome (ex.: roteamento por país).
	Resolve(name string) (Gateway, error)
}
