package user

import "context"

// Repository é a PORTA de persistência do agregado User, implementada pela
// infraestrutura.
type Repository interface {
	// Save persiste um usuário novo. Falha com ErrEmailTaken se o e-mail já existir.
	Save(ctx context.Context, u *User) error
	// Update grava mutações de um usuário existente (concorrência otimista).
	Update(ctx context.Context, u *User) error
	// FindByID hidrata um usuário pelo id (ErrNotFound se não existir).
	FindByID(ctx context.Context, id UserID) (*User, error)
	// FindByEmail resolve um usuário pelo e-mail (ErrNotFound se não existir).
	FindByEmail(ctx context.Context, email Email) (*User, error)
	// List devolve todos os usuários (read model simples para o demo).
	List(ctx context.Context) ([]*User, error)
}
