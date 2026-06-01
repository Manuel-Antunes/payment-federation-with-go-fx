package application

import (
	"go.uber.org/fx"

	"github.com/example/payment-federation/internal/payments/application/bus"
	"github.com/example/payment-federation/internal/payments/application/command"
	"github.com/example/payment-federation/internal/payments/application/query"
)

// Module é o módulo de APLICAÇÃO do bounded context de pagamentos. Provê os
// use-cases (command/query) e REGISTRA seus handlers nos barramentos de CQRS:
// commands e events no Watermill, queries no QueryBus genérico. Inclui a SAGA
// que dispara o pagamento ao criar um pedido.
var Module = fx.Module("payments/application",
	fx.Provide(
		// Use-cases (command).
		command.NewProcessPaymentHandler,
		command.NewRefundPaymentHandler,
		command.NewCreateOrderHandler,
		// Use-cases (query).
		query.NewGetPaymentHandler,
		query.NewGetPaymentByKeyHandler,
		query.NewGetOrderHandler,
		query.NewGetOrderByKeyHandler,
	),

	// Registro de commands, queries e events (saga incluída).
	fx.Invoke(bus.RegisterCommands),
	fx.Invoke(bus.RegisterQueries),
	fx.Invoke(bus.RegisterEvents),
)
