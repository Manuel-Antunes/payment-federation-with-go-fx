package infrastructure

import (
	"context"
	"database/sql"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/example/payment-federation/internal/user/application/port"
	"github.com/example/payment-federation/internal/user/domain/user"
	"github.com/example/payment-federation/internal/user/infrastructure/adapters"
	entx "github.com/example/payment-federation/internal/user/infrastructure/ent"
	"github.com/example/payment-federation/internal/user/infrastructure/persistence"
)

// Module é o módulo de INFRAESTRUTURA do bounded context de usuário. Provê o
// client ent (sobre o *sql.DB compartilhado), o repositório ent e o gerador de
// id; e migra o schema do módulo no start.
var Module = fx.Module("user/infrastructure",
	fx.Provide(
		newEntClient,
		// Repository (porta) <- EntRepository.
		fx.Annotate(
			persistence.NewEntRepository,
			fx.As(new(user.Repository)),
		),
		// IDGenerator (porta) <- UUIDGenerator.
		func() port.IDGenerator { return adapters.UUIDGenerator{} },
	),
	fx.Invoke(migrate),
)

// newEntClient cria o client ent do módulo por cima do *sql.DB compartilhado.
func newEntClient(db *sql.DB) *entx.Client {
	drv := entsql.OpenDB(dialect.Postgres, db)
	return entx.NewClient(entx.Driver(drv))
}

// migrate cria/atualiza as tabelas do módulo de usuário no start (auto-migração
// do ent — a recomendação do getting-started; em produção, usar Atlas).
func migrate(lc fx.Lifecycle, client *entx.Client, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("migrando schema do módulo user")
			return client.Schema.Create(ctx)
		},
	})
}
