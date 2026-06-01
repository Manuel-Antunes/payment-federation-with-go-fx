package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// User é o schema ent da tabela de usuários. Id é string (gerado no domínio,
// ex.: "usr_<uuid>"); o e-mail é único.
type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Immutable(),
		field.String("name").NotEmpty(),
		field.String("email").NotEmpty().Unique(),
		field.Time("created_at").Immutable(),
		field.Time("updated_at"),
		field.Int("version"),
	}
}

func (User) Edges() []ent.Edge { return nil }
