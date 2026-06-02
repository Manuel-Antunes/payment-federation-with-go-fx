package bus

import (
	"github.com/example/payment-federation/internal/shared/cqrs"
	"github.com/example/payment-federation/internal/user/application/command"
)

// RegisterCommands registra os commands do módulo de usuário. Cada
// cqrs.RegisterCommand fia o dispatch genérico + o handler no Watermill — o
// módulo não toca no Watermill. Os use-cases implementam CommandHandler[C, R],
// então são passados diretamente.
func RegisterCommands(
	bus *cqrs.CommandBus,
	create *command.CreateUserHandler,
	update *command.UpdateUserHandler,
) error {
	if err := cqrs.RegisterCommand(bus, "create_user", create); err != nil {
		return err
	}
	return cqrs.RegisterCommand(bus, "update_user", update)
}
