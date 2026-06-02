// Package cqrs é um mediator de CQRS genérico e COMPARTILHADO entre módulos.
//
// Expõe um CommandBus (escrita), um QueryBus (leitura) e um EventBus (pub/sub).
// Cada barramento é um registry que roteia a mensagem ao seu handler pela TYPE
// da mensagem (reflect.Type).
//
// Como o Go não permite type parameters em métodos, o registro e a leitura
// tipados são FUNÇÕES genéricas (RegisterCommand/RegisterQuery/Ask/...) sobre um
// bus que guarda handlers "type-erased". Isso dá segurança de tipo no registro e
// no dispatch, mantendo o barramento global e adaptável a qualquer módulo.
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
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	wmcqrs "github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// ---------------------------------------------------------------------------
// Command side — o contrato é uma interface genérica: o handler trata um comando
// C e devolve um resultado R (+ erro). Os use-cases já têm essa forma
// (Handle(ctx, C) (R, error)), então passam direto, sem closures de adaptação.
//
// Há dois modos de execução:
//   - Dispatch: fire-and-forget — despacha e ignora R (princípio do CQRS, o
//     resultado é observado via read model). Usado por sagas/integrações.
//   - ExecuteMutationSync: request-reply — despacha pelo barramento do Watermill
//     e DEVOLVE o R do handler, via um canal correlacionado por id. Para os casos
//     em que a API realmente precisa do dado produzido pela escrita.
// ---------------------------------------------------------------------------

// CommandHandler trata um comando C e devolve um resultado R.
type CommandHandler[C any, R any] interface {
	Handle(ctx context.Context, cmd C) (R, error)
}

// correlationMetadataKey é a chave na metadata da mensagem que carrega o id de
// correlação do request-reply. O middleware do router (ver shared/messaging) a
// promove para o context que o handler recebe.
const correlationMetadataKey = "cqrs_correlation_id"

type correlationCtxKey struct{}

// CorrelationMetadataKey expõe a chave de metadata para o middleware de transporte.
func CorrelationMetadataKey() string { return correlationMetadataKey }

// ContextWithCorrelation injeta o id de correlação no context (usado pelo
// middleware ao consumir a mensagem).
func ContextWithCorrelation(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationCtxKey{}, id)
}

func correlationFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(correlationCtxKey{}).(string)
	return id, ok && id != ""
}

// reply transporta o resultado type-erased de um command de volta ao chamador.
type reply struct {
	val any
	err error
}

// CommandBus roteia comandos para seus handlers, pela type do comando. Carrega o
// transporte e o processor do Watermill (injetados no construtor), de modo que
// RegisterCommand não precisa recebê-los — os módulos só conhecem o
// *cqrs.CommandBus. Mantém também um registry de canais de resposta para o
// request-reply síncrono (ExecuteMutationSync).
type CommandBus struct {
	handlers  map[reflect.Type]func(ctx context.Context, cmd any) error
	transport *wmcqrs.CommandBus
	processor *wmcqrs.CommandProcessor

	mu      sync.Mutex
	replies map[string]chan reply
}

func NewCommandBus(transport *wmcqrs.CommandBus, processor *wmcqrs.CommandProcessor) *CommandBus {
	return &CommandBus{
		handlers:  make(map[reflect.Type]func(context.Context, any) error),
		transport: transport,
		processor: processor,
		replies:   make(map[string]chan reply),
	}
}

// RegisterCommand fia um comando C (resultado R) de ponta a ponta, numa única
// chamada:
//
//   - DISPATCH: ao despachar C, ele é encaminhado para o CommandBus do Watermill
//     via Send (block-until-ack).
//   - HANDLER: registra o use-case como handler de C no CommandProcessor; após
//     executá-lo, se houver um id de correlação no context (request-reply),
//     entrega o resultado/erro no canal de resposta correspondente.
//
// A política de Ack-em-erro (evitar reentrega infinita no block-until-ack) fica
// centralizada num middleware do router (ver internal/shared/messaging) — por
// isso o erro de negócio do `Dispatch` é observado via read model, enquanto o
// `ExecuteMutationSync` recebe o erro diretamente pelo canal de resposta.
func RegisterCommand[C any, R any](
	bus *CommandBus,
	handlerName string,
	handler CommandHandler[C, R],
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
	// CommandProcessor do Watermill -> invoca o use-case e responde se houver
	// um request-reply pendente (id de correlação no context).
	return bus.processor.AddHandlers(
		wmcqrs.NewCommandHandler(handlerName, func(ctx context.Context, cmd *C) error {
			res, err := handler.Handle(ctx, *cmd)
			if id, ok := correlationFromContext(ctx); ok {
				bus.fulfill(id, reply{val: res, err: err})
			}
			return err
		}),
	)
}

// Dispatch despacha um comando (fire-and-forget) para o handler do seu tipo.
func (b *CommandBus) Dispatch(ctx context.Context, cmd any) error {
	h, ok := b.handlers[reflect.TypeOf(cmd)]
	if !ok {
		return fmt.Errorf("cqrs: nenhum handler de comando para %T", cmd)
	}
	return h(ctx, cmd)
}

// ExecuteMutationSync despacha o comando C pelo barramento do Watermill e devolve
// o resultado R produzido pelo handler — request-reply correlacionado por id,
// entregue via canal. Para os casos em que a escrita realmente precisa retornar
// dados. O context controla o timeout/cancelamento da espera.
func ExecuteMutationSync[C any, R any](ctx context.Context, b *CommandBus, cmd C) (R, error) {
	var zero R

	id := watermill.NewUUID()
	ch := make(chan reply, 1) // buffered: fulfill nunca bloqueia
	b.registerReply(id, ch)
	defer b.unregisterReply(id)

	// Anexa o id de correlação na metadata da mensagem; o middleware do router o
	// promove para o context que o handler recebe.
	err := b.transport.SendWithModifiedMessage(ctx, &cmd, func(m *message.Message) error {
		m.Metadata.Set(correlationMetadataKey, id)
		return nil
	})
	if err != nil {
		return zero, err
	}

	select {
	case r := <-ch:
		if r.err != nil {
			return zero, r.err
		}
		out, ok := r.val.(R)
		if !ok {
			return zero, fmt.Errorf("cqrs: resultado de %T tem tipo inesperado %T", cmd, r.val)
		}
		return out, nil
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

func (b *CommandBus) registerReply(id string, ch chan reply) {
	b.mu.Lock()
	b.replies[id] = ch
	b.mu.Unlock()
}

func (b *CommandBus) unregisterReply(id string) {
	b.mu.Lock()
	delete(b.replies, id)
	b.mu.Unlock()
}

func (b *CommandBus) fulfill(id string, r reply) {
	b.mu.Lock()
	ch, ok := b.replies[id]
	b.mu.Unlock()
	if ok {
		ch <- r // canal buffered(1): entrega sem bloquear
	}
}
