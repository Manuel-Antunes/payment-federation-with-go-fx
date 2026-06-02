package persistence

import (
	"context"

	"github.com/example/payment-federation/internal/payments/domain/order"
	entx "github.com/example/payment-federation/internal/payments/infrastructure/ent"
	entorder "github.com/example/payment-federation/internal/payments/infrastructure/ent/order"
)

// OrderEntRepository é o adaptador ent da porta order.Repository.
type OrderEntRepository struct {
	client *entx.Client
	mapper orderSchema
}

func NewOrderEntRepository(client *entx.Client) *OrderEntRepository {
	return &OrderEntRepository{client: client}
}

var _ order.Repository = (*OrderEntRepository)(nil)

func (r *OrderEntRepository) Save(ctx context.Context, o *order.Order) error {
	_, err := r.mapper.toCreate(r.client, o).Save(ctx)
	if entx.IsConstraintError(err) {
		return order.ErrAlreadyExists // idempotency_key único violado
	}
	return err
}

func (r *OrderEntRepository) FindByID(ctx context.Context, id order.OrderID) (*order.Order, error) {
	row, err := r.client.Order.Get(ctx, id.String())
	if entx.IsNotFound(err) {
		return nil, order.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.mapper.toDomain(row), nil
}

func (r *OrderEntRepository) FindByIDs(ctx context.Context, ids []order.OrderID) ([]*order.Order, error) {
	raw := make([]string, 0, len(ids))
	for _, id := range ids {
		raw = append(raw, id.String())
	}
	rows, err := r.client.Order.Query().Where(entorder.IDIn(raw...)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*order.Order, 0, len(rows))
	for _, row := range rows {
		out = append(out, r.mapper.toDomain(row))
	}
	return out, nil
}

// FindByIdempotencyKey resolve o pedido pela sua chave EFETIVA: encontra pela
// idempotency_key (quando o cliente forneceu uma) OU pelo próprio id (quando não
// forneceu — a chave efetiva passa a ser o id). Assim o mesmo lookup serve tanto
// para a chave custom quanto para o id do pedido (read-after-write do createOrder
// relê pelo id).
func (r *OrderEntRepository) FindByIdempotencyKey(ctx context.Context, key string) (*order.Order, error) {
	row, err := r.client.Order.Query().
		Where(entorder.Or(
			entorder.IdempotencyKey(key),
			entorder.ID(key),
		)).
		First(ctx)
	if entx.IsNotFound(err) {
		return nil, order.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.mapper.toDomain(row), nil
}
