package infrastructure

import (
	"go.uber.org/fx"

	"github.com/example/payment-federation/internal/user/application/port"
	"github.com/example/payment-federation/internal/user/domain/user"
	"github.com/example/payment-federation/internal/user/infrastructure/adapters"
	"github.com/example/payment-federation/internal/user/infrastructure/persistence"
)

// Module é o módulo de INFRAESTRUTURA do bounded context de usuário. Provê os
// adaptadores concretos e as referências de IoC das portas da aplicação:
// repositórios, geradores de id, etc.
var Module = fx.Module("user/infrastructure",
	fx.Provide(
		// Repository (porta) <- MemoryRepository (adaptador concreto).
		fx.Annotate(
			persistence.NewMemoryRepository,
			fx.As(new(user.Repository)),
		),
		// IDGenerator (porta) <- UUIDGenerator.
		func() port.IDGenerator { return adapters.UUIDGenerator{} },
	),
)
