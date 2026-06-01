package bus

import (
	"context"

	"github.com/example/payment-federation/internal/payments/application/command"
	"github.com/example/payment-federation/internal/shared/cqrs"
)

// As mensagens de comando são os próprios DTOs de command (campos exportados,
// serializáveis). Cada cqrs.RegisterCommand fia, de uma vez, o dispatch genérico
// e o handler no CommandProcessor do Watermill — o módulo não toca no Watermill.

// RegisterCommands registra os commands do módulo de pagamentos.
func RegisterCommands(
	bus *cqrs.CommandBus,
	process *command.ProcessPaymentHandler,
	refund *command.RefundPaymentHandler,
	createOrder *command.CreateOrderHandler,
) error {
	if err := cqrs.RegisterCommand(bus, "process_payment",
		func(ctx context.Context, cmd command.ProcessPayment) error {
			_, err := process.Handle(ctx, cmd)
			return err
		}); err != nil {
		return err
	}
	if err := cqrs.RegisterCommand(bus, "refund_payment",
		func(ctx context.Context, cmd command.RefundPayment) error {
			_, err := refund.Handle(ctx, cmd)
			return err
		}); err != nil {
		return err
	}
	return cqrs.RegisterCommand(bus, "create_order",
		func(ctx context.Context, cmd command.CreateOrder) error {
			_, err := createOrder.Handle(ctx, cmd)
			return err
		})
}
