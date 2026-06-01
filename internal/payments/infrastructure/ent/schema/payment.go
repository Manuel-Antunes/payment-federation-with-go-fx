package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Payment é o schema ent da tabela de pagamentos. idempotency_key é único.
type Payment struct {
	ent.Schema
}

func (Payment) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Immutable(),
		field.String("idempotency_key").NotEmpty().Unique().Immutable(),
		field.String("customer_id").NotEmpty().Immutable(),
		field.Int64("amount_cents"),
		field.String("currency").Immutable(),
		field.Int64("refunded_cents").Default(0),
		field.String("status"),
		field.String("gateway_ref").Default(""),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
		field.Int("version"),
	}
}

func (Payment) Edges() []ent.Edge { return nil }
