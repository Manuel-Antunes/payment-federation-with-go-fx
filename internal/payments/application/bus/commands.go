package bus

import (
	"github.com/example/payment-federation/internal/payments/application/command"
	"github.com/example/payment-federation/internal/shared/cqrs"
)

// As mensagens de comando são os próprios DTOs de command (campos exportados,
// serializáveis). Cada cqrs.RegisterCommand fia, de uma vez, o dispatch genérico
// e o handler no CommandProcessor do Watermill — o módulo não toca no Watermill.
// Os use-cases já implementam o contrato CommandHandler[C, R] (Handle(ctx, C)
// (R, error)), então são passados diretamente, sem closures de adaptação.

// RegisterCommands registra os commands do módulo de pagamentos.
func RegisterCommands(
	bus *cqrs.CommandBus,
	process *command.ProcessPaymentHandler,
	refund *command.RefundPaymentHandler,
	createOrder *command.CreateOrderHandler,
) error {
	if err := cqrs.RegisterCommand(bus, "process_payment", process); err != nil {
		return err
	}
	if err := cqrs.RegisterCommand(bus, "refund_payment", refund); err != nil {
		return err
	}
	return cqrs.RegisterCommand(bus, "create_order", createOrder)
}
