package query

import "github.com/example/payment-federation/internal/user/domain/user"

// UserView é o read model (DTO plano) do usuário, compartilhado pelas queries.
type UserView struct {
	ID    string
	Name  string
	Email string
}

func viewFrom(u *user.User) UserView {
	return UserView{
		ID:    u.ID().String(),
		Name:  u.Name().String(),
		Email: u.Email().String(),
	}
}
