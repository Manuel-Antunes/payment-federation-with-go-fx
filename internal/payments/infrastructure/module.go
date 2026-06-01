package infrastructure

import (
	"context"
	"database/sql"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/example/payment-federation/internal/payments/application/port"
	"github.com/example/payment-federation/internal/payments/domain/order"
	"github.com/example/payment-federation/internal/payments/domain/payment"
	"github.com/example/payment-federation/internal/payments/infrastructure/adapters"
	entx "github.com/example/payment-federation/internal/payments/infrastructure/ent"
	"github.com/example/payment-federation/internal/payments/infrastructure/gateway"
	"github.com/example/payment-federation/internal/payments/infrastructure/messaging"
	"github.com/example/payment-federation/internal/payments/infrastructure/persistence"
)

// Module é o módulo de INFRAESTRUTURA do bounded context de pagamentos. Provê o
// client ent do módulo (sobre o *sql.DB compartilhado), os repositórios ent,
// gateways, geradores de id e publishers de evento; e migra o schema no start.
var Module = fx.Module("payments/infrastructure",
	fx.Provide(
		newEntClient,
		// Repository de pagamento (porta) <- PaymentEntRepository.
		fx.Annotate(
			persistence.NewPaymentEntRepository,
			fx.As(new(payment.Repository)),
		),
		// Repository de pedido (porta) <- OrderEntRepository.
		fx.Annotate(
			persistence.NewOrderEntRepository,
			fx.As(new(order.Repository)),
		),

		// Registry de gateways concreto (consumido pelo admin/HTTP)...
		newGatewayRegistry,
		// ...e exposto como a porta payment.GatewayProvider.
		func(r *gateway.Registry) payment.GatewayProvider { return r },

		// IDGenerators (portas) <- UUID.
		func() port.IDGenerator { return adapters.UUIDGenerator{} },
		func() port.OrderIDGenerator { return adapters.OrderUUIDGenerator{} },

		// EventPublishers (portas) <- Watermill (EventBus).
		fx.Annotate(
			messaging.NewWatermillPublisher,
			fx.As(new(port.EventPublisher)),
		),
		fx.Annotate(
			messaging.NewWatermillOrderPublisher,
			fx.As(new(port.OrderEventPublisher)),
		),
	),
	fx.Invoke(migrate),
)

// newEntClient cria o client ent do módulo por cima do *sql.DB compartilhado.
func newEntClient(db *sql.DB) *entx.Client {
	drv := entsql.OpenDB(dialect.Postgres, db)
	return entx.NewClient(entx.Driver(drv))
}

// migrate cria/atualiza as tabelas do módulo de pagamentos no start.
func migrate(lc fx.Lifecycle, client *entx.Client, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("migrando schema do módulo payments")
			return client.Schema.Create(ctx)
		},
	})
}

// newGatewayRegistry registra os provedores disponíveis e define o ativo
// inicial. Novos provedores podem ser trocados em runtime via /admin/gateway.
func newGatewayRegistry(log *zap.Logger) *gateway.Registry {
	r := gateway.NewRegistry()
	r.Register(gateway.NewMockGateway())
	r.Register(gateway.NewStripeGateway("sk_test_xxx", 1_000_000)) // recusa > R$10.000,00
	_ = r.SetActive("mock")                                        // provedor padrão
	log.Info("gateways registrados", zap.Strings("available", r.Available()))
	return r
}
