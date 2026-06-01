package bus

import (
	"context"

	"github.com/example/payment-federation/internal/shared/cqrs"
	"github.com/example/payment-federation/internal/user/application/command"
)

// RegisterCommands registra os commands do módulo de usuário. Cada
// cqrs.RegisterCommand fia o dispatch genérico + o handler no Watermill — o
// módulo não toca no Watermill.
func RegisterCommands(
	bus *cqrs.CommandBus,
	create *command.CreateUserHandler,
	update *command.UpdateUserHandler,
) error {
	if err := cqrs.RegisterCommand(bus, "create_user",
		func(ctx context.Context, cmd command.CreateUser) error {
			_, err := create.Handle(ctx, cmd)
			return err
		}); err != nil {
		return err
	}
	return cqrs.RegisterCommand(bus, "update_user",
		func(ctx context.Context, cmd command.UpdateUser) error {
			_, err := update.Handle(ctx, cmd)
			return err
		})
}
