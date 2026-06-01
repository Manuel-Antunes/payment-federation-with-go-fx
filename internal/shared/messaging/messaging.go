// Package messaging é o kernel de mensageria COMPARTILHADO entre módulos. Monta
// o backbone de CQRS sobre o Watermill — CommandBus, CommandProcessor, EventBus
// e EventProcessor — todos sobre um mesmo router e um mesmo Pub/Sub.
//
// Não conhece nada de pagamentos: expõe construtores genéricos que cada módulo
// usa para registrar seus próprios commands/events.
//
// O transporte padrão é o GoChannel (in-memory) configurado com
// BlockPublishUntilSubscriberAck: Send/Publish bloqueiam até o handler dar Ack.
// Assim a escrita assíncrona se comporta de forma síncrona o suficiente para o
// read-after-write da API GraphQL. Trocar por Kafka/NATS/SQS é só substituir o
// Pub/Sub — domínio e aplicação não mudam.
package messaging

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Marshaler (de)serializa commands e events. O tópico é derivado do nome do
// tipo Go (mesma struct no publish e no subscribe => os tópicos batem).
func Marshaler() cqrs.CommandEventMarshaler {
	return cqrs.JSONMarshaler{
		GenerateName: cqrs.NamedStruct(cqrs.StructName),
	}
}

// NewPubSub cria o transporte in-memory (GoChannel). Implementa tanto
// message.Publisher quanto message.Subscriber — e DEVE ser a MESMA instância
// para publicar e assinar (o estado vive no processo).
func NewPubSub(logger watermill.LoggerAdapter) *gochannel.GoChannel {
	return gochannel.NewGoChannel(gochannel.Config{
		// Send/Publish bloqueia até o subscriber dar Ack: o efeito do command
		// já está aplicado quando Send retorna (viabiliza o read-after-write).
		BlockPublishUntilSubscriberAck: true,
	}, logger)
}

// NewRouter cria o router que executa os handlers (consumers) do barramento.
//
// Middlewares (ordem = de fora para dentro):
//   - ackOnError: faz Ack mesmo quando o handler devolve erro/panic. Com o
//     GoChannel em block-until-ack, um Nack causaria reentrega infinita e
//     travaria o Send; aqui logamos o erro e seguimos. Centraliza a política
//     para que os handlers possam devolver erros normalmente.
//   - Recoverer: converte panics em erros (capturados então pelo ackOnError).
func NewRouter(logger watermill.LoggerAdapter) (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, err
	}
	router.AddMiddleware(ackOnError(logger))
	router.AddMiddleware(middleware.Recoverer)
	return router, nil
}

// ackOnError loga o erro do handler e faz Ack (devolve nil), evitando o loop de
// reentrega do block-until-ack. O resultado de negócio é observado via read
// model (read-after-write), não pelo erro do barramento.
func ackOnError(logger watermill.LoggerAdapter) message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			out, err := h(msg)
			if err != nil {
				logger.Error("handler error (acking to avoid redelivery)", err, watermill.LogFields{
					"message_uuid": msg.UUID,
				})
				return out, nil
			}
			return out, nil
		}
	}
}

// --- Estratégia de tópicos: usamos o próprio nome do command/event. ---

func commandPublishTopic(p cqrs.CommandBusGeneratePublishTopicParams) (string, error) {
	return p.CommandName, nil
}

func commandSubscribeTopic(p cqrs.CommandProcessorGenerateSubscribeTopicParams) (string, error) {
	return p.CommandName, nil
}

func eventPublishTopic(p cqrs.GenerateEventPublishTopicParams) (string, error) {
	return p.EventName, nil
}

func eventSubscribeTopic(p cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
	return p.EventName, nil
}

// NewCommandBus cria o barramento de COMANDOS (lado de despacho da escrita).
func NewCommandBus(pub message.Publisher, logger watermill.LoggerAdapter) (*cqrs.CommandBus, error) {
	return cqrs.NewCommandBusWithConfig(pub, cqrs.CommandBusConfig{
		GeneratePublishTopic: commandPublishTopic,
		Marshaler:            Marshaler(),
		Logger:               logger,
	})
}

// NewCommandProcessor cria o processor que assina os tópicos de comando e roda
// os handlers no router. Cada módulo chama AddHandlers para registrar os seus.
func NewCommandProcessor(router *message.Router, sub message.Subscriber, logger watermill.LoggerAdapter) (*cqrs.CommandProcessor, error) {
	return cqrs.NewCommandProcessorWithConfig(router, cqrs.CommandProcessorConfig{
		GenerateSubscribeTopic: commandSubscribeTopic,
		SubscriberConstructor: func(cqrs.CommandProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return sub, nil
		},
		Marshaler: Marshaler(),
		Logger:    logger,
	})
}

// NewEventBus cria o barramento de EVENTOS (lado de publicação).
func NewEventBus(pub message.Publisher, logger watermill.LoggerAdapter) (*cqrs.EventBus, error) {
	return cqrs.NewEventBusWithConfig(pub, cqrs.EventBusConfig{
		GeneratePublishTopic: eventPublishTopic,
		Marshaler:            Marshaler(),
		Logger:               logger,
	})
}

// NewEventProcessor cria o processor que assina os tópicos de evento e roda os
// handlers (projeções/integrações) no router.
func NewEventProcessor(router *message.Router, sub message.Subscriber, logger watermill.LoggerAdapter) (*cqrs.EventProcessor, error) {
	return cqrs.NewEventProcessorWithConfig(router, cqrs.EventProcessorConfig{
		GenerateSubscribeTopic: eventSubscribeTopic,
		SubscriberConstructor: func(cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return sub, nil
		},
		Marshaler: Marshaler(),
		Logger:    logger,
	})
}

// RunRouter pendura o ciclo de vida do router no fx: sobe num goroutine no
// OnStart (esperando ficar pronto) e fecha no OnStop. Todos os AddHandlers dos
// módulos rodam na fase de Invoke do fx, ANTES de qualquer OnStart — então os
// consumers já estão registrados quando o router sobe.
func RunRouter(lc fx.Lifecycle, router *message.Router, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				// context.Background(): o router só para via Close() (OnStop).
				if err := router.Run(context.Background()); err != nil {
					log.Error("message router stopped", zap.Error(err))
				}
			}()
			select {
			case <-router.Running():
				log.Info("message router up")
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		OnStop: func(ctx context.Context) error {
			log.Info("message router shutting down")
			return router.Close()
		},
	})
}
