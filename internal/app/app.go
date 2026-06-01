// Package app é a COMPOSIÇÃO da aplicação (composition root): une o kernel
// compartilhado, os bounded contexts (user, payments), a interface única
// (GraphQL/HTTP) e os bridges cross-module.
package app

import (
	"go.uber.org/fx"

	"github.com/example/payment-federation/internal/interfaces/graph"
	httpiface "github.com/example/payment-federation/internal/interfaces/http"
	paymentsport "github.com/example/payment-federation/internal/payments/application/port"
	"github.com/example/payment-federation/internal/payments"
	"github.com/example/payment-federation/internal/shared"
	"github.com/example/payment-federation/internal/user"
)

// Module agrega tudo o que o servidor precisa para subir.
var Module = fx.Options(
	// Kernel compartilhado (relógio, mensageria/CQRS).
	shared.Module,

	// Bounded contexts.
	user.Module,
	payments.Module,

	// Bridges cross-module (composição): valida o cliente do pedido consultando
	// o módulo de usuário, sem acoplar os módulos.
	fx.Provide(
		fx.Annotate(
			newCustomerDirectory,
			fx.As(new(paymentsport.CustomerDirectory)),
		),
	),

	// Interface única (subgraph GraphQL + servidor HTTP).
	fx.Provide(
		graph.NewResolver,
		httpiface.NewFiberApp,
	),
)
