// Package shared é o módulo COMPARTILHADO da aplicação: agrega os provedores
// de capacidades técnicas reutilizáveis entre bounded contexts (relógio e o
// backbone de mensageria CQRS do Watermill). Não conhece pagamentos.
package shared

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/fx"

	"github.com/example/payment-federation/internal/shared/clock"
	"github.com/example/payment-federation/internal/shared/cqrs"
	"github.com/example/payment-federation/internal/shared/messaging"
)

// Module expõe o kernel compartilhado para o fx.
var Module = fx.Module("shared",
	fx.Provide(
		// Clock (porta genérica) <- SystemClock.
		func() clock.Clock { return clock.SystemClock{} },

		// Logger do Watermill (adaptado do zap).
		messaging.NewWatermillLogger,

		// Pub/Sub in-memory: UMA instância exposta como Publisher e Subscriber
		// (o GoChannel guarda estado no processo, então tem de ser a mesma).
		fx.Annotate(
			messaging.NewPubSub,
			fx.As(new(message.Publisher)),
			fx.As(new(message.Subscriber)),
		),

		// Router + os quatro componentes de CQRS do Watermill.
		messaging.NewRouter,
		messaging.NewCommandBus,
		messaging.NewCommandProcessor,
		messaging.NewEventBus,
		messaging.NewEventProcessor,

		// Mediators de CQRS genéricos (registry por tipo), consumidos pelos
		// módulos. CommandBus e EventBus recebem o transporte/processors do
		// Watermill no construtor (fx injeta), encapsulando essa dependência —
		// os módulos só lidam com os tipos do pacote cqrs. O QueryBus é síncrono.
		cqrs.NewCommandBus,
		cqrs.NewEventBus,
		cqrs.NewQueryBus,

		// Publisher GENÉRICO de eventos de domínio (lado de publicação do
		// EventBus do Watermill) — usado por todos os módulos via For/Commit.
		cqrs.NewEventPublisher,
	),

	// Sobe/desce o router no lifecycle do fx (após os módulos registrarem seus
	// handlers nos processors, ainda na fase de Invoke).
	fx.Invoke(messaging.RunRouter),
)
