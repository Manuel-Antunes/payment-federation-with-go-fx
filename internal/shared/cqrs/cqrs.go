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
	"fmt"
	"reflect"

	wmcqrs "github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// typeKey devolve a reflect.Type estática de T (funciona para structs, ponteiros
// e interfaces), usada como chave do registry.
func typeKey[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// ---------------------------------------------------------------------------
// Command side — comandos não retornam dados (apenas erro). O resultado é
// observado pelo read model (read-after-write).
// ---------------------------------------------------------------------------

// CommandHandler trata um comando do tipo C.
type CommandHandler[C any] func(ctx context.Context, cmd C) error

// CommandBus roteia comandos para seus handlers, pela type do comando. Carrega
// o transporte e o processor do Watermill (injetados no construtor), de modo que
// RegisterCommand não precisa recebê-los — os módulos só conhecem o *cqrs.CommandBus.
type CommandBus struct {
	handlers  map[reflect.Type]func(ctx context.Context, cmd any) error
	transport *wmcqrs.CommandBus
	processor *wmcqrs.CommandProcessor
}

func NewCommandBus(transport *wmcqrs.CommandBus, processor *wmcqrs.CommandProcessor) *CommandBus {
	return &CommandBus{
		handlers:  make(map[reflect.Type]func(context.Context, any) error),
		transport: transport,
		processor: processor,
	}
}

// RegisterCommand fia um comando C de ponta a ponta, numa única chamada:
//
//   - DISPATCH: ao despachar C no CommandBus genérico, ele é encaminhado para o
//     CommandBus do Watermill via Send (block-until-ack).
//   - HANDLER: registra `handle` (o use-case) como handler de C no
//     CommandProcessor do Watermill.
//
// Substitui o antigo par RegisterCommandDispatch + RegisterCommandHandlers. Os
// erros de negócio podem ser retornados normalmente por `handle`: a política de
// Ack-em-erro (evitar reentrega infinita no block-until-ack) fica centralizada
// num middleware do router (ver internal/shared/messaging).
func RegisterCommand[C any](
	bus *CommandBus,
	handlerName string,
	handle CommandHandler[C],
) error {
	t := typeKey[C]()
	if _, exists := bus.handlers[t]; exists {
		return fmt.Errorf("cqrs: command já registrado para %s", t)
	}
	// Dispatch genérico -> Send no transporte Watermill.
	bus.handlers[t] = func(ctx context.Context, cmd any) error {
		c := cmd.(C)
		return bus.transport.Send(ctx, &c)
	}
	// CommandProcessor do Watermill -> invoca o use-case.
	return bus.processor.AddHandlers(
		wmcqrs.NewCommandHandler(handlerName, func(ctx context.Context, cmd *C) error {
			return handle(ctx, *cmd)
		}),
	)
}

// Dispatch despacha um comando para o handler registrado para o seu tipo.
func (b *CommandBus) Dispatch(ctx context.Context, cmd any) error {
	h, ok := b.handlers[reflect.TypeOf(cmd)]
	if !ok {
		return fmt.Errorf("cqrs: nenhum handler de comando para %T", cmd)
	}
	return h(ctx, cmd)
}

// ---------------------------------------------------------------------------
// Event side — consumo de eventos (projeções, sagas) sobre o EventProcessor do
// Watermill. RegisterEvent encapsula o wmcqrs aqui, espelhando RegisterCommand:
// os módulos só fornecem o nome e a função de tratamento, sem tocar no Watermill.
// ---------------------------------------------------------------------------

// EventBus carrega o EventProcessor do Watermill (injetado no construtor), de
// modo que RegisterEvent não precisa recebê-lo — os módulos só conhecem o
// *cqrs.EventBus.
type EventBus struct {
	processor *wmcqrs.EventProcessor
}

func NewEventBus(processor *wmcqrs.EventProcessor) *EventBus {
	return &EventBus{processor: processor}
}

// EventHandler trata um evento do tipo E (recebido por ponteiro, como no Watermill).
type EventHandler[E any] func(ctx context.Context, event *E) error

// RegisterEvent registra um handler tipado para o evento E no EventProcessor.
// O tópico/roteamento é derivado do tipo E pelo marshaler (mesma struct no
// publish e no subscribe => topics batem).
func RegisterEvent[E any](
	bus *EventBus,
	handlerName string,
	handle EventHandler[E],
) error {
	return bus.processor.AddHandlers(
		wmcqrs.NewEventHandler(handlerName, func(ctx context.Context, e *E) error {
			return handle(ctx, e)
		}),
	)
}

// ---------------------------------------------------------------------------
// Query side — queries retornam dados.
// ---------------------------------------------------------------------------

// QueryHandler trata uma query do tipo Q e devolve um resultado R.
type QueryHandler[Q any, R any] func(ctx context.Context, query Q) (R, error)

// QueryBus roteia queries para seus handlers, pela type da query.
type QueryBus struct {
	handlers map[reflect.Type]func(ctx context.Context, query any) (any, error)
}

func NewQueryBus() *QueryBus {
	return &QueryBus{handlers: make(map[reflect.Type]func(context.Context, any) (any, error))}
}

// RegisterQuery vincula um handler tipado ao tipo de query Q (resultado R).
func RegisterQuery[Q any, R any](b *QueryBus, handler QueryHandler[Q, R]) error {
	t := typeKey[Q]()
	if _, exists := b.handlers[t]; exists {
		return fmt.Errorf("cqrs: query handler já registrado para %s", t)
	}
	b.handlers[t] = func(ctx context.Context, query any) (any, error) {
		return handler(ctx, query.(Q))
	}
	return nil
}

// Ask despacha uma query e devolve o resultado já tipado (R).
func Ask[Q any, R any](ctx context.Context, b *QueryBus, query Q) (R, error) {
	var zero R
	h, ok := b.handlers[typeKey[Q]()]
	if !ok {
		return zero, fmt.Errorf("cqrs: nenhum handler de query para %T", query)
	}
	res, err := h(ctx, query)
	if err != nil {
		return zero, err
	}
	out, ok := res.(R)
	if !ok {
		return zero, fmt.Errorf("cqrs: resultado de %T tem tipo inesperado %T", query, res)
	}
	return out, nil
}
