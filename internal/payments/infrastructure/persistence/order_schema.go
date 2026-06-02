package persistence

import (
	"github.com/example/payment-federation/internal/payments/domain/order"
	entx "github.com/example/payment-federation/internal/payments/infrastructure/ent"
)

// orderRow é alias da linha ent (evita colisão com o agregado "Order").
type orderRow = entx.Order

// orderSchema casa a linha ent com o agregado de domínio, EMBUTINDO ambos, e
// concentra a conversão ent <-> domínio de pedido.
type orderSchema struct {
	*orderRow
	*order.Order
}

func (orderSchema) toDomain(r *entx.Order) *order.Order {
	return order.FromSnapshot(order.Snapshot{
		ID:             r.ID,
		IdempotencyKey: r.IdempotencyKey,
		CustomerID:     r.CustomerID,
		AmountCents:    r.AmountCents,
		Currency:       r.Currency,
		Status:         r.Status,
		CreatedAt:      r.CreatedAt,
		Version:        r.Version,
	})
}

func (orderSchema) toCreate(client *entx.Client, o *order.Order) *entx.OrderCreate {
	s := o.ToSnapshot()
	return client.Order.Create().
		SetID(s.ID).
		SetNillableIdempotencyKey(s.IdempotencyKey).
		SetCustomerID(s.CustomerID).
		SetAmountCents(s.AmountCents).
		SetCurrency(s.Currency).
		SetStatus(s.Status).
		SetCreatedAt(s.CreatedAt).
		SetVersion(s.Version)
}
