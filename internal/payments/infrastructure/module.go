package infrastructure

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/example/payment-federation/internal/payments/application/port"
	"github.com/example/payment-federation/internal/payments/domain/order"
	"github.com/example/payment-federation/internal/payments/domain/payment"
	"github.com/example/payment-federation/internal/payments/infrastructure/adapters"
	"github.com/example/payment-federation/internal/payments/infrastructure/gateway"
	"github.com/example/payment-federation/internal/payments/infrastructure/messaging"
	"github.com/example/payment-federation/internal/payments/infrastructure/persistence"
)

// Module é o módulo de INFRAESTRUTURA do bounded context de pagamentos. Provê
// os adaptadores concretos e as referências de IoC das portas da aplicação:
// repositórios, gateways/serviços, geradores de id e publishers de evento.
var Module = fx.Module("payments/infrastructure",
	fx.Provide(
		// Repository de pagamento (porta) <- MemoryRepository.
		fx.Annotate(
			persistence.NewMemoryRepository,
			fx.As(new(payment.Repository)),
		),
		// Repository de pedido (porta) <- OrderMemoryRepository.
		fx.Annotate(
			persistence.NewOrderMemoryRepository,
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
)

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
