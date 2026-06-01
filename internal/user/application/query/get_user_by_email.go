package query

import (
	"context"

	"github.com/example/payment-federation/internal/user/domain/user"
)

// GetUserByEmail resolve um usuário pelo e-mail (read-after-write de createUser).
type GetUserByEmail struct{ Email string }

type GetUserByEmailHandler struct{ repo user.Repository }

func NewGetUserByEmailHandler(repo user.Repository) *GetUserByEmailHandler {
	return &GetUserByEmailHandler{repo: repo}
}

func (h *GetUserByEmailHandler) Handle(ctx context.Context, q GetUserByEmail) (*user.User, error) {
	email, err := user.NewEmail(q.Email)
	if err != nil {
		return nil, err
	}
	return h.repo.FindByEmail(ctx, email)
}
