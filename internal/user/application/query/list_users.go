package query

import (
	"context"

	"github.com/example/payment-federation/internal/user/domain/user"
)

// ListUsers devolve todos os usuários (read model simples para o demo).
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
