package command

import (
	"context"

	"github.com/example/payment-federation/internal/shared/clock"
	"github.com/example/payment-federation/internal/shared/cqrs"
	"github.com/example/payment-federation/internal/shared/domain"
	"github.com/example/payment-federation/internal/user/application/event"
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
	repo   user.Repository
	ids    port.IDGenerator
	clock  clock.Clock
	events cqrs.EventPublisher
}

func NewCreateUserHandler(repo user.Repository, ids port.IDGenerator, clk clock.Clock, events cqrs.EventPublisher) *CreateUserHandler {
	return &CreateUserHandler{repo: repo, ids: ids, clock: clk, events: events}
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

	// Publica UserCreated APÓS persistir.
	_ = h.events.For(u).Commit(ctx, func(e domain.DomainEvent) (any, bool) {
		uc, ok := e.(user.UserCreated)
		if !ok {
			return nil, false
		}
		return event.UserIntegrationFrom(uc), true
	})
	return CreateUserResult{UserID: u.ID().String()}, nil
}
