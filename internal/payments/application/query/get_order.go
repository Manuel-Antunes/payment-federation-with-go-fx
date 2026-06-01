package query

import (
	"context"

	"github.com/example/payment-federation/internal/payments/domain/order"
)

// OrderView é o read model (DTO plano) do pedido.
type OrderView struct {
	ID          string
	CustomerID  string
	AmountCents int64
	Currency    string
	Status      string
}

func orderViewFrom(o *order.Order) OrderView {
	return OrderView{
		ID:          o.ID().String(),
		CustomerID:  o.CustomerID(),
		AmountCents: o.Amount().AmountCents(),
		Currency:    string(o.Amount().Currency()),
		Status:      string(o.Status()),
	}
}

// GetOrder resolve um pedido por id.
type GetOrder struct{ OrderID string }

type GetOrderHandler struct{ repo order.Repository }

func NewGetOrderHandler(repo order.Repository) *GetOrderHandler { return &GetOrderHandler{repo: repo} }

func (h *GetOrderHandler) Handle(ctx context.Context, q GetOrder) (OrderView, error) {
	o, err := h.repo.FindByID(ctx, order.OrderID(q.OrderID))
	if err != nil {
		return OrderView{}, err
	}
	return orderViewFrom(o), nil
}

// GetOrderByKey resolve um pedido pela chave de idempotência (read-after-write
// da mutation createOrder, já que o id é gerado dentro do use-case).
type GetOrderByKey struct{ IdempotencyKey string }

type GetOrderByKeyHandler struct{ repo order.Repository }

func NewGetOrderByKeyHandler(repo order.Repository) *GetOrderByKeyHandler {
	return &GetOrderByKeyHandler{repo: repo}
}

func (h *GetOrderByKeyHandler) Handle(ctx context.Context, q GetOrderByKey) (OrderView, error) {
	o, err := h.repo.FindByIdempotencyKey(ctx, q.IdempotencyKey)
	if err != nil {
		return OrderView{}, err
	}
	return orderViewFrom(o), nil
}
