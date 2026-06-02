// Package cqrs é um mediator de CQRS genérico e COMPARTILHADO entre módulos.
//
// Expõe um CommandBus (escrita, sem retorno de dados — princípio do CQRS) e um
// QueryBus (leitura, retorna dados). Cada barramento é um registry que roteia a
// mensagem ao seu handler pela TYPE da mensagem (reflect.Type).
//
// Como o Go não permite type parameters em métodos, o registro e a leitura
// tipados são FUNÇÕES genéricas (RegisterCommand/RegisterQuery/Ask) sobre um bus
// que guarda handlers "type-erased". Isso dá segurança de tipo no registro e no
// dispatch, mantendo o barramento global e adaptável a qualquer módulo.
//
// No lado de COMANDO o barramento é apoiado pelo Watermill: um único
// RegisterCommand fia, de uma vez, o dispatch (Dispatch -> CommandBus.Send) e o
// handler no CommandProcessor do Watermill. (O lado de QUERY é síncrono e não
// depende do Watermill.)
package cqrs

import (
	"context"

	wmcqrs "github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// ---------------------------------------------------------------------------
// Event side — PUBLICAÇÃO (Publish) e CONSUMO (RegisterEvent) de eventos sobre o
// EventBus/EventProcessor do Watermill. Ambos os lados ficam encapsulados aqui:
// os módulos só conhecem o *cqrs.EventBus e nunca tocam no Watermill.
// ---------------------------------------------------------------------------

// EventBus envelopa os dois lados de eventos do Watermill (injetados no
// construtor): o EventBus (publicação) e o EventProcessor (consumo). Assim
// Publish e RegisterEvent não precisam recebê-los — os módulos só conhecem o
// *cqrs.EventBus.
type EventBus struct {
	publisher *wmcqrs.EventBus
	processor *wmcqrs.EventProcessor
}

func NewEventBus(publisher *wmcqrs.EventBus, processor *wmcqrs.EventProcessor) *EventBus {
	return &EventBus{publisher: publisher, processor: processor}
}

// Publish publica um evento no barramento, delegando ao EventBus do Watermill.
// É o ponto único de publicação do nosso CQRS — hoje envelopa o Watermill; no
// futuro pode rotear de forma genérica (outro transporte, outbox transacional,
// fan-out, etc.) sem que os chamadores mudem.
func (b *EventBus) Publish(ctx context.Context, event any) error {
	return b.publisher.Publish(ctx, event)
}

// EventHandler trata um evento do tipo E (recebido por ponteiro, como no
// Watermill). É uma interface, espelhando CommandHandler/QueryHandler: os
// consumidores (projeções, sagas) são tipos com método Handle, passados direto.
type EventHandler[E any] interface {
	Handle(ctx context.Context, event *E) error
}

// RegisterEvent registra um handler tipado para o evento E no EventProcessor.
// O tópico/roteamento é derivado do tipo E pelo marshaler (mesma struct no
// publish e no subscribe => topics batem).
func RegisterEvent[E any](
	bus *EventBus,
	handlerName string,
	handler EventHandler[E],
) error {
	return bus.processor.AddHandlers(
		wmcqrs.NewEventHandler(handlerName, func(ctx context.Context, e *E) error {
			return handler.Handle(ctx, e)
		}),
	)
}
