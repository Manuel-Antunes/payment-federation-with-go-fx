package query

import (
	"context"

	"github.com/example/payment-federation/internal/payments/domain/order"
)

// GetOrder resolve um pedido por id (devolve o agregado de domínio).
type GetOrder struct{ OrderID string }

type GetOrderHandler struct{ repo order.Repository }

func NewGetOrderHandler(repo order.Repository) *GetOrderHandler { return &GetOrderHandler{repo: repo} }

func (h *GetOrderHandler) Handle(ctx context.Context, q GetOrder) (*order.Order, error) {
	return h.repo.FindByID(ctx, order.OrderID(q.OrderID))
}

// GetOrdersByIDs é a query de LEITURA EM LOTE — resolve N pedidos numa só
// chamada. É o que o DataLoader de pedidos despacha para fazer o batch.
type GetOrdersByIDs struct{ IDs []string }

type GetOrdersByIDsHandler struct{ repo order.Repository }

func NewGetOrdersByIDsHandler(repo order.Repository) *GetOrdersByIDsHandler {
	return &GetOrdersByIDsHandler{repo: repo}
}

func (h *GetOrdersByIDsHandler) Handle(ctx context.Context, q GetOrdersByIDs) ([]*order.Order, error) {
	ids := make([]order.OrderID, 0, len(q.IDs))
	for _, id := range q.IDs {
		ids = append(ids, order.OrderID(id))
	}
	return h.repo.FindByIDs(ctx, ids)
}

// GetOrderByKey resolve um pedido pela chave de idempotência (read-after-write
// da mutation createOrder, já que o id é gerado dentro do use-case).
type GetOrderByKey struct{ IdempotencyKey string }

type GetOrderByKeyHandler struct{ repo order.Repository }

func NewGetOrderByKeyHandler(repo order.Repository) *GetOrderByKeyHandler {
	return &GetOrderByKeyHandler{repo: repo}
}

func (h *GetOrderByKeyHandler) Handle(ctx context.Context, q GetOrderByKey) (*order.Order, error) {
	return h.repo.FindByIdempotencyKey(ctx, q.IdempotencyKey)
}
