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
)

// typeKey devolve a reflect.Type estática de T (funciona para structs, ponteiros
// e interfaces), usada como chave do registry.
func typeKey[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// ---------------------------------------------------------------------------
// Query side — queries retornam dados.
// ---------------------------------------------------------------------------

// QueryHandler trata uma query do tipo Q e devolve um resultado R.

type QueryHandler[Q any, R any] interface {
	Handle(ctx context.Context, query Q) (R, error)
}

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
		return handler.Handle(ctx, query.(Q))
	}
	return nil
}

// ExecuteQuery despacha uma query e devolve o resultado já tipado (R).
func ExecuteQuery[Q any, R any](ctx context.Context, b *QueryBus, query Q) (R, error) {
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
