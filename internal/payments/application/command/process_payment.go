package command

import (
	"context"
	"errors"

	"github.com/example/payment-federation/internal/payments/application/event"
	"github.com/example/payment-federation/internal/payments/application/port"
	"github.com/example/payment-federation/internal/payments/domain/payment"
	"github.com/example/payment-federation/internal/shared/clock"
	"github.com/example/payment-federation/internal/shared/cqrs"
	"github.com/example/payment-federation/internal/shared/domain"
)

// ProcessPayment é o COMMAND (lado de escrita do CQRS): cria, autoriza e
// captura um pagamento numa única operação idempotente.
type ProcessPayment struct {
	IdempotencyKey string
	CustomerID     string
	AmountCents    int64
	Currency       string
	// GatewayName opcional: roteia para um provedor específico em runtime.
	// Vazio => usa o gateway ativo do provider.
	GatewayName string
}

// ProcessPaymentResult é o DTO devolvido (sem expor o agregado).
type ProcessPaymentResult struct {
	PaymentID string
	Status    string
	Replayed  bool // true quando a idempotência devolveu um pagamento já existente
}

// ProcessPaymentHandler orquestra domínio + gateway. NOTA: toda a regra de
// negócio roda no agregado em memória; a PERSISTÊNCIA acontece só no final,
// depois que todas as invariantes foram aplicadas.
type ProcessPaymentHandler struct {
	repo     payment.Repository
	gateways payment.GatewayProvider
	ids      port.IDGenerator
	clock    clock.Clock
	events   cqrs.EventPublisher
}

func NewProcessPaymentHandler(
	repo payment.Repository,
	gateways payment.GatewayProvider,
	ids port.IDGenerator,
	clock clock.Clock,
	events cqrs.EventPublisher,
) *ProcessPaymentHandler {
	return &ProcessPaymentHandler{repo: repo, gateways: gateways, ids: ids, clock: clock, events: events}
}

func (h *ProcessPaymentHandler) Handle(ctx context.Context, cmd ProcessPayment) (ProcessPaymentResult, error) {
	// 1) Construção dos Value Objects — validação de entrada é regra de domínio.
	key, err := payment.NewIdempotencyKey(cmd.IdempotencyKey)
	if err != nil {
		return ProcessPaymentResult{}, err
	}
	cur, err := payment.NewCurrency(cmd.Currency)
	if err != nil {
		return ProcessPaymentResult{}, err
	}
	amount, err := payment.NewMoney(cmd.AmountCents, cur)
	if err != nil {
		return ProcessPaymentResult{}, err
	}

	// 2) IDEMPOTÊNCIA: se a chave já produziu um pagamento, devolve-o (replay).
	if existing, err := h.repo.FindByIdempotencyKey(ctx, key); err == nil {
		return ProcessPaymentResult{
			PaymentID: existing.ID().String(),
			Status:    string(existing.Status()),
			Replayed:  true,
		}, nil
	} else if !errors.Is(err, payment.ErrNotFound) {
		return ProcessPaymentResult{}, err
	}

	// 3) Cria o agregado (fábrica aplica invariantes de criação).
	clk := payment.Clock(h.clock.Now)
	p, err := payment.NewPayment(h.ids.NewID(), key, cmd.CustomerID, amount, clk)
	if err != nil {
		return ProcessPaymentResult{}, err
	}

	// 4) Resolve o gateway em RUNTIME e autoriza.
	gw, err := h.selectGateway(cmd.GatewayName)
	if err != nil {
		return ProcessPaymentResult{}, err
	}

	authRes, gwErr := gw.Authorize(ctx, payment.AuthorizeInput{
		PaymentID:      p.ID(),
		IdempotencyKey: key,
		CustomerID:     cmd.CustomerID,
		Amount:         amount,
	})

	// 5) Aplica o resultado do gateway COMO REGRA DE DOMÍNIO (ainda em memória).
	switch {
	case gwErr != nil:
		_ = p.Fail("gateway error: "+gwErr.Error(), clk)
	case !authRes.Approved:
		_ = p.Fail("declined: "+authRes.DeclineReason, clk)
	default:
		if err := p.Authorize(authRes.GatewayRef, clk); err != nil {
			return ProcessPaymentResult{}, err
		}
		if err := p.Capture(clk); err != nil {
			return ProcessPaymentResult{}, err
		}
		if err := gw.Capture(ctx, payment.CaptureInput{GatewayRef: authRes.GatewayRef, Amount: amount}); err != nil {
			return ProcessPaymentResult{}, err
		}
	}

	// 6) PERSISTÊNCIA — só agora, com o agregado em estado final e válido.
	if err := h.repo.Save(ctx, p); err != nil {
		// Corrida: outra requisição com a mesma chave persistiu primeiro.
		if errors.Is(err, payment.ErrAlreadyExists) {
			if existing, e := h.repo.FindByIdempotencyKey(ctx, key); e == nil {
				return ProcessPaymentResult{PaymentID: existing.ID().String(), Status: string(existing.Status()), Replayed: true}, nil
			}
		}
		return ProcessPaymentResult{}, err
	}

	// 7) Publica eventos após persistir — o mapper anexa o snapshot do agregado.
	_ = h.events.For(p).Commit(ctx, func(e domain.DomainEvent) (any, bool) {
		return event.PaymentIntegrationFrom(p, e), true
	})

	return ProcessPaymentResult{PaymentID: p.ID().String(), Status: string(p.Status())}, nil
}

func (h *ProcessPaymentHandler) selectGateway(name string) (payment.Gateway, error) {
	if name == "" {
		return h.gateways.Active(), nil
	}
	return h.gateways.Resolve(name)
}
