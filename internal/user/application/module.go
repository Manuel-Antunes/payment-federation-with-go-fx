package application

import (
	"go.uber.org/fx"

	"github.com/example/payment-federation/internal/user/application/bus"
	"github.com/example/payment-federation/internal/user/application/command"
	"github.com/example/payment-federation/internal/user/application/query"
)

// Module é o módulo de APLICAÇÃO do bounded context de usuário. Provê os
// use-cases (command/query) e REGISTRA seus handlers nos barramentos de CQRS
// (commands no Watermill, queries no QueryBus genérico). Não conhece a infra —
// depende apenas das portas e do kernel compartilhado.
var Module = fx.Module("user/application",
	fx.Provide(
		// Use-cases (command).
		command.NewCreateUserHandler,
		command.NewUpdateUserHandler,
		// Use-cases (query).
		query.NewGetUserHandler,
		query.NewGetUserByEmailHandler,
		query.NewListUsersHandler,
		query.NewExistsHandler,
	),

	// Registro de commands e queries.
	fx.Invoke(bus.RegisterCommands),
	fx.Invoke(bus.RegisterQueries),
)
