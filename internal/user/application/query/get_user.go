package query

import (
	"context"

	"github.com/example/payment-federation/internal/user/domain/user"
)

// GetUser resolve um usuário por id.
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
