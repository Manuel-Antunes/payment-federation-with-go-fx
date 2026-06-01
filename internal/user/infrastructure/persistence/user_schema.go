package persistence

import (
	entx "github.com/example/payment-federation/internal/user/infrastructure/ent"
	"github.com/example/payment-federation/internal/user/domain/user"
)

// userRow é alias da linha gerada pelo ent. Aliás para evitar a colisão de nome
// de campo embutido com o agregado de domínio (ambos se chamam "User").
type userRow = entx.User

// userSchema casa a linha ent (entx.User) com o agregado de domínio (user.User),
// EMBUTINDO ambos, e concentra a conversão entre os dois mundos. É o ponto único
// de mapeamento ent <-> domínio para usuário.
type userSchema struct {
	*userRow
	*user.User
}

// toDomain reidrata o agregado de domínio a partir da linha ent (via snapshot).
func (userSchema) toDomain(r *entx.User) *user.User {
	return user.FromSnapshot(user.Snapshot{
		ID:        r.ID,
		Name:      r.Name,
		Email:     r.Email,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		Version:   r.Version,
	})
}

// toCreate monta o builder de inserção ent a partir do agregado.
func (userSchema) toCreate(client *entx.Client, u *user.User) *entx.UserCreate {
	s := u.ToSnapshot()
	return client.User.Create().
		SetID(s.ID).
		SetName(s.Name).
		SetEmail(s.Email).
		SetCreatedAt(s.CreatedAt).
		SetUpdatedAt(s.UpdatedAt).
		SetVersion(s.Version)
}
