package command

import (
	"context"

	"github.com/example/payment-federation/internal/shared/clock"
	"github.com/example/payment-federation/internal/user/domain/user"
)

// UpdateUser é o command de atualização. Campos nil = "não alterar".
type UpdateUser struct {
	UserID string
	Name   *string
	Email  *string
}

type UpdateUserResult struct {
	UserID string
}

type UpdateUserHandler struct {
	repo  user.Repository
	clock clock.Clock
}

func NewUpdateUserHandler(repo user.Repository, clk clock.Clock) *UpdateUserHandler {
	return &UpdateUserHandler{repo: repo, clock: clk}
}

func (h *UpdateUserHandler) Handle(ctx context.Context, cmd UpdateUser) (UpdateUserResult, error) {
	u, err := h.repo.FindByID(ctx, user.UserID(cmd.UserID))
	if err != nil {
		return UpdateUserResult{}, err
	}

	if cmd.Name != nil {
		name, err := user.NewName(*cmd.Name)
		if err != nil {
			return UpdateUserResult{}, err
		}
		u.Rename(name, h.clock.Now)
	}
	if cmd.Email != nil {
		email, err := user.NewEmail(*cmd.Email)
		if err != nil {
			return UpdateUserResult{}, err
		}
		u.ChangeEmail(email, h.clock.Now)
	}

	if err := h.repo.Update(ctx, u); err != nil {
		return UpdateUserResult{}, err
	}
	return UpdateUserResult{UserID: u.ID().String()}, nil
}
