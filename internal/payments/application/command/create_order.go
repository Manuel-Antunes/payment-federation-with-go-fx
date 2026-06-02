package command

import (
	"context"

	"github.com/example/payment-federation/internal/payments/application/event"
	"github.com/example/payment-federation/internal/payments/application/port"
	"github.com/example/payment-federation/internal/payments/domain/order"
	"github.com/example/payment-federation/internal/payments/domain/payment"
	"github.com/example/payment-federation/internal/shared/clock"
	"github.com/example/payment-federation/internal/shared/cqrs"
	"github.com/example/payment-federation/internal/shared/domain"
)

// CreateOrder é o command de criação de pedido. Exige um cliente (CustomerID).
type CreateOrder struct {
	IdempotencyKey *string
	CustomerID     string
	AmountCents    int64
	Currency       string
}

type CreateOrderResult struct {
	OrderID string
}

// CreateOrderHandler cria o pedido. Valida que o cliente existe (via
// CustomerDirectory), aplica as invariantes do agregado, persiste e publica
// OrderCreated — que o barramento usa para disparar o pagamento (saga).
type CreateOrderHandler struct {
	repo      order.Repository
	customers port.CustomerDirectory
	ids       port.OrderIDGenerator
	clock     clock.Clock
	events    cqrs.EventPublisher
}

func NewCreateOrderHandler(
	repo order.Repository,
	customers port.CustomerDirectory,
	ids port.OrderIDGenerator,
	clk clock.Clock,
	events cqrs.EventPublisher,
) *CreateOrderHandler {
	return &CreateOrderHandler{repo: repo, customers: customers, ids: ids, clock: clk, events: events}
}

func (h *CreateOrderHandler) Handle(ctx context.Context, cmd CreateOrder) (CreateOrderResult, error) {
	var idempotencyKey *payment.IdempotencyKey
	if cmd.IdempotencyKey != nil {
		key, err := payment.NewIdempotencyKey(*cmd.IdempotencyKey)
		if err != nil {
			return CreateOrderResult{}, err
		}
		idempotencyKey = &key
	}

	cur, err := payment.NewCurrency(cmd.Currency)
	if err != nil {
		return CreateOrderResult{}, err
	}
	amount, err := payment.NewMoney(cmd.AmountCents, cur)
	if err != nil {
		return CreateOrderResult{}, err
	}

	// Um pedido EXIGE um cliente existente (validação cross-module via porta).
	if cmd.CustomerID == "" {
		return CreateOrderResult{}, order.ErrEmptyCustomer
	}
	exists, err := h.customers.Exists(ctx, cmd.CustomerID)
	if err != nil {
		return CreateOrderResult{}, err
	}
	if !exists {
		return CreateOrderResult{}, order.ErrCustomerUnknown
	}

	o, err := order.NewOrder(h.ids.NewID(), idempotencyKey, cmd.CustomerID, amount, h.clock.Now)
	if err != nil {
		return CreateOrderResult{}, err
	}
	if err := h.repo.Save(ctx, o); err != nil {
		return CreateOrderResult{}, err
	}

	// Publica OrderCreated APÓS persistir — dispara a saga de pagamento.
	_ = h.events.For(o).Commit(ctx, func(e domain.DomainEvent) (any, bool) {
		oc, ok := e.(order.OrderCreated)
		if !ok {
			return nil, false
		}
		return event.OrderIntegrationFrom(oc), true
	})
	return CreateOrderResult{OrderID: o.ID().String()}, nil
}
