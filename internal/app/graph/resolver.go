package graph

import (
	"github.com/example/payment-federation/internal/shared/cqrs"
)

//go:generate go run github.com/99designs/gqlgen generate

// Resolver é a raiz de dependências injetada pelo fx. Ele NÃO contém regra de
// negócio — apenas traduz GraphQL <-> os barramentos de CQRS genéricos
// (CommandBus para escrita, QueryBus para leitura).
type Resolver struct {
	Commands *cqrs.CommandBus
	Queries  *cqrs.QueryBus
	Payments *PaymentHub
}

func NewResolver(commands *cqrs.CommandBus, queries *cqrs.QueryBus, payments *PaymentHub) *Resolver {
	return &Resolver{Commands: commands, Queries: queries, Payments: payments}
}
