package graph

import (
	"context"

	"github.com/example/payment-federation/internal/interfaces/graph/model"
	pquery "github.com/example/payment-federation/internal/payments/application/query"
	"github.com/example/payment-federation/internal/shared/cqrs"
	uquery "github.com/example/payment-federation/internal/user/application/query"
)

// Este arquivo NÃO é gerado pelo gqlgen. Concentra os helpers de read-after-write
// e o mapeamento read model -> model GraphQL. Todos despacham pelo QueryBus
// genérico compartilhado (qualquer módulo registrado responde).

// ---- Payment --------------------------------------------------------------

func (r *Resolver) load(ctx context.Context, id string) (*model.Payment, error) {
	view, err := cqrs.Ask[pquery.GetPayment, pquery.PaymentView](ctx, r.Queries, pquery.GetPayment{PaymentID: id})
	if err != nil {
		return nil, err
	}
	return toGraphPayment(view), nil
}

// loadByKey relê pelo idempotencyKey (o id do pagamento é gerado no use-case).
func (r *Resolver) loadByKey(ctx context.Context, idempotencyKey string) (*model.Payment, error) {
	view, err := cqrs.Ask[pquery.GetPaymentByKey, pquery.PaymentView](ctx, r.Queries, pquery.GetPaymentByKey{IdempotencyKey: idempotencyKey})
	if err != nil {
		return nil, err
	}
	return toGraphPayment(view), nil
}

func toGraphPayment(v pquery.PaymentView) *model.Payment {
	p := &model.Payment{
		ID:            v.ID,
		CustomerID:    v.CustomerID,
		AmountCents:   v.AmountCents,
		Currency:      v.Currency,
		RefundedCents: v.RefundedCents,
		Status:        v.Status,
	}
	if v.GatewayRef != "" {
		ref := v.GatewayRef
		p.GatewayRef = &ref
	}
	return p
}

// ---- Order ----------------------------------------------------------------
//
// A leitura por id (order(id), FindOrderByID) passa pelo DataLoader (ver
// internal/interfaces/graph/dataloader). Aqui fica só o read-after-write por
// chave de idempotência, que não é por id.

// loadOrderByKey relê o pedido pela chave de idempotência (read-after-write).
func (r *Resolver) loadOrderByKey(ctx context.Context, idempotencyKey string) (*model.Order, error) {
	view, err := cqrs.Ask[pquery.GetOrderByKey, pquery.OrderView](ctx, r.Queries, pquery.GetOrderByKey{IdempotencyKey: idempotencyKey})
	if err != nil {
		return nil, err
	}
	return toGraphOrder(view), nil
}

func toGraphOrder(v pquery.OrderView) *model.Order {
	return &model.Order{
		ID:          v.ID,
		CustomerID:  v.CustomerID,
		AmountCents: v.AmountCents,
		Currency:    v.Currency,
		Status:      v.Status,
	}
}

// ---- User -----------------------------------------------------------------

func (r *Resolver) loadUser(ctx context.Context, id string) (*model.User, error) {
	view, err := cqrs.Ask[uquery.GetUser, uquery.UserView](ctx, r.Queries, uquery.GetUser{UserID: id})
	if err != nil {
		return nil, err
	}
	return toGraphUser(view), nil
}

// loadUserByEmail relê o usuário pelo e-mail (read-after-write de createUser).
func (r *Resolver) loadUserByEmail(ctx context.Context, email string) (*model.User, error) {
	view, err := cqrs.Ask[uquery.GetUserByEmail, uquery.UserView](ctx, r.Queries, uquery.GetUserByEmail{Email: email})
	if err != nil {
		return nil, err
	}
	return toGraphUser(view), nil
}

func (r *Resolver) listUsers(ctx context.Context) ([]*model.User, error) {
	views, err := cqrs.Ask[uquery.ListUsers, []uquery.UserView](ctx, r.Queries, uquery.ListUsers{})
	if err != nil {
		return nil, err
	}
	out := make([]*model.User, 0, len(views))
	for _, v := range views {
		out = append(out, toGraphUser(v))
	}
	return out, nil
}

func toGraphUser(v uquery.UserView) *model.User {
	return &model.User{ID: v.ID, Name: v.Name, Email: v.Email}
}
