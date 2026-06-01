// Package user é o DOMÍNIO PURO do bounded context de Usuário. Sem dependências
// de infra: o agregado valida invariantes em memória e só é persistido quando
// já está válido.
package user

import "time"

// Clock é injetável para testes determinísticos.
type Clock func() time.Time

// User é o Aggregate Root. Campos privados: o estado só muda por métodos de
// comando que validam ANTES de mutar.
type User struct {
	id        UserID
	name      Name
	email     Email
	createdAt time.Time
	updatedAt time.Time
	version   int
}

// NewUser é a fábrica do agregado. Recebe um id já gerado (a porta de
// identidade fica fora do domínio puro).
func NewUser(id UserID, name Name, email Email, now Clock) (*User, error) {
	if id.IsZero() {
		return nil, ErrNotFound
	}
	t := now()
	return &User{
		id:        id,
		name:      name,
		email:     email,
		createdAt: t,
		updatedAt: t,
		version:   1,
	}, nil
}

// Rename altera o nome de exibição.
func (u *User) Rename(name Name, now Clock) {
	u.name = name
	u.touch(now)
}

// ChangeEmail troca o e-mail.
func (u *User) ChangeEmail(email Email, now Clock) {
	u.email = email
	u.touch(now)
}

func (u *User) touch(now Clock) {
	u.updatedAt = now()
	u.version++
}

// ---- Getters (read-only) ---------------------------------------------------

func (u *User) ID() UserID           { return u.id }
func (u *User) Name() Name           { return u.name }
func (u *User) Email() Email         { return u.email }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) UpdatedAt() time.Time { return u.updatedAt }
func (u *User) Version() int         { return u.version }

// ---- Snapshot (persistência) ----------------------------------------------

// Snapshot é a representação plana usada pelo repositório.
type Snapshot struct {
	ID        string
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
	Version   int
}

func (u *User) ToSnapshot() Snapshot {
	return Snapshot{
		ID:        u.id.String(),
		Name:      u.name.String(),
		Email:     u.email.String(),
		CreatedAt: u.createdAt,
		UpdatedAt: u.updatedAt,
		Version:   u.version,
	}
}

// FromSnapshot reidrata o agregado sem revalidar (já era válido ao gravar).
func FromSnapshot(s Snapshot) *User {
	return &User{
		id:        UserID(s.ID),
		name:      Name(s.Name),
		email:     Email(s.Email),
		createdAt: s.CreatedAt,
		updatedAt: s.UpdatedAt,
		version:   s.Version,
	}
}
