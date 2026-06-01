package query

import (
	"context"

	"github.com/example/payment-federation/internal/user/domain/user"
)

// UserView é o read model (DTO plano) do usuário.
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

// --- GetUser (por id) ---

type GetUser struct{ UserID string }

type GetUserHandler struct{ repo user.Repository }

func NewGetUserHandler(repo user.Repository) *GetUserHandler { return &GetUserHandler{repo: repo} }

func (h *GetUserHandler) Handle(ctx context.Context, q GetUser) (UserView, error) {
	u, err := h.repo.FindByID(ctx, user.UserID(q.UserID))
	if err != nil {
		return UserView{}, err
	}
	return viewFrom(u), nil
}

// --- GetUserByEmail (read-after-write de createUser) ---

type GetUserByEmail struct{ Email string }

type GetUserByEmailHandler struct{ repo user.Repository }

func NewGetUserByEmailHandler(repo user.Repository) *GetUserByEmailHandler {
	return &GetUserByEmailHandler{repo: repo}
}

func (h *GetUserByEmailHandler) Handle(ctx context.Context, q GetUserByEmail) (UserView, error) {
	email, err := user.NewEmail(q.Email)
	if err != nil {
		return UserView{}, err
	}
	u, err := h.repo.FindByEmail(ctx, email)
	if err != nil {
		return UserView{}, err
	}
	return viewFrom(u), nil
}

// --- ListUsers ---

type ListUsers struct{}

type ListUsersHandler struct{ repo user.Repository }

func NewListUsersHandler(repo user.Repository) *ListUsersHandler { return &ListUsersHandler{repo: repo} }

func (h *ListUsersHandler) Handle(ctx context.Context, _ ListUsers) ([]UserView, error) {
	users, err := h.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]UserView, 0, len(users))
	for _, u := range users {
		views = append(views, viewFrom(u))
	}
	return views, nil
}

// Exists indica se um usuário existe pelo id. Usado pelo módulo de pagamentos
// (via bridge) para validar o cliente de um pedido — sem acoplar os módulos.
type Exists struct{ UserID string }

type ExistsHandler struct{ repo user.Repository }

func NewExistsHandler(repo user.Repository) *ExistsHandler { return &ExistsHandler{repo: repo} }

func (h *ExistsHandler) Handle(ctx context.Context, q Exists) (bool, error) {
	_, err := h.repo.FindByID(ctx, user.UserID(q.UserID))
	if err == user.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
