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

func (r *OrderEntRepository) FindByIdempotencyKey(ctx context.Context, key string) (*order.Order, error) {
	row, err := r.client.Order.Query().Where(entorder.IdempotencyKey(key)).Only(ctx)
	if entx.IsNotFound(err) {
		return nil, order.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.mapper.toDomain(row), nil
}
