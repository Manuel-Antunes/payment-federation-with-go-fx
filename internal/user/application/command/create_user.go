package command

import (
	"context"

	"github.com/example/payment-federation/internal/shared/clock"
	"github.com/example/payment-federation/internal/user/application/port"
	"github.com/example/payment-federation/internal/user/domain/user"
)

// CreateUser é o command de criação de usuário (lado de escrita do CQRS).
type CreateUser struct {
	Name  string
	Email string
}

type CreateUserResult struct {
	UserID string
}

type CreateUserHandler struct {
	repo  user.Repository
	ids   port.IDGenerator
	clock clock.Clock
}

func NewCreateUserHandler(repo user.Repository, ids port.IDGenerator, clk clock.Clock) *CreateUserHandler {
	return &CreateUserHandler{repo: repo, ids: ids, clock: clk}
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUser) (CreateUserResult, error) {
	name, err := user.NewName(cmd.Name)
	if err != nil {
		return CreateUserResult{}, err
	}
	email, err := user.NewEmail(cmd.Email)
	if err != nil {
		return CreateUserResult{}, err
	}

	u, err := user.NewUser(h.ids.NewID(), name, email, h.clock.Now)
	if err != nil {
		return CreateUserResult{}, err
	}
	if err := h.repo.Save(ctx, u); err != nil {
		return CreateUserResult{}, err
	}
	return CreateUserResult{UserID: u.ID().String()}, nil
}
