package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Order é o schema ent da tabela de pedidos. idempotency_key é único.
type Order struct {
	ent.Schema
}

func (Order) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Immutable(),
		field.String("idempotency_key").NotEmpty().Unique().Immutable(),
		field.String("customer_id").NotEmpty().Immutable(),
		field.Int64("amount_cents").Immutable(),
		field.String("currency").Immutable(),
		field.String("status"),
		field.Time("created_at").Immutable(),
		field.Int("version"),
	}
}

func (Order) Edges() []ent.Edge { return nil }
