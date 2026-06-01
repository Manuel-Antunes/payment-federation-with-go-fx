package graph

import (
	"context"

	"github.com/example/payment-federation/internal/app/graph/model"
	pquery "github.com/example/payment-federation/internal/payments/application/query"
	"github.com/example/payment-federation/internal/payments/domain/order"
	"github.com/example/payment-federation/internal/payments/domain/payment"
	"github.com/example/payment-federation/internal/shared/cqrs"
	uquery "github.com/example/payment-federation/internal/user/application/query"
	"github.com/example/payment-federation/internal/user/domain/user"
)

// Este arquivo NÃO é gerado pelo gqlgen. As queries devolvem os AGREGADOS de
// domínio; aqui (a borda) traduzimos para os models GraphQL. Todos despacham
// pelo QueryBus genérico compartilhado.

// ---- Payment --------------------------------------------------------------

func (r *Resolver) load(ctx context.Context, id string) (*model.Payment, error) {
	p, err := cqrs.Ask[pquery.GetPayment, *payment.Payment](ctx, r.Queries, pquery.GetPayment{PaymentID: id})
	if err != nil {
		return nil, err
	}
	return toGraphPayment(p), nil
}

// loadByKey relê pelo idempotencyKey (o id do pagamento é gerado no use-case).
func (r *Resolver) loadByKey(ctx context.Context, idempotencyKey string) (*model.Payment, error) {
	p, err := cqrs.Ask[pquery.GetPaymentByKey, *payment.Payment](ctx, r.Queries, pquery.GetPaymentByKey{IdempotencyKey: idempotencyKey})
	if err != nil {
		return nil, err
	}
	return toGraphPayment(p), nil
}

func toGraphPayment(p *payment.Payment) *model.Payment {
	m := &model.Payment{
		ID:            p.ID().String(),
		CustomerID:    p.CustomerID(),
		AmountCents:   p.Amount().AmountCents(),
		Currency:      string(p.Amount().Currency()),
		RefundedCents: p.Refunded().AmountCents(),
		Status:        string(p.Status()),
	}
	if ref := p.GatewayRef(); !ref.IsEmpty() {
		s := ref.String()
		m.GatewayRef = &s
	}
	return m
}

// ---- Order ----------------------------------------------------------------
//
// A leitura por id (order(id), FindOrderByID) passa pelo DataLoader (ver
// internal/app/graph/dataloader). Aqui fica só o read-after-write por chave.

// loadOrderByKey relê o pedido pela chave de idempotência (read-after-write).
func (r *Resolver) loadOrderByKey(ctx context.Context, idempotencyKey string) (*model.Order, error) {
	o, err := cqrs.Ask[pquery.GetOrderByKey, *order.Order](ctx, r.Queries, pquery.GetOrderByKey{IdempotencyKey: idempotencyKey})
	if err != nil {
		return nil, err
	}
	return toGraphOrder(o), nil
}

func toGraphOrder(o *order.Order) *model.Order {
	return &model.Order{
		ID:          o.ID().String(),
		CustomerID:  o.CustomerID(),
		AmountCents: o.Amount().AmountCents(),
		Currency:    string(o.Amount().Currency()),
		Status:      string(o.Status()),
	}
}

// ---- User -----------------------------------------------------------------

func (r *Resolver) loadUser(ctx context.Context, id string) (*model.User, error) {
	u, err := cqrs.Ask[uquery.GetUser, *user.User](ctx, r.Queries, uquery.GetUser{UserID: id})
	if err != nil {
		return nil, err
	}
	return toGraphUser(u), nil
}

// loadUserByEmail relê o usuário pelo e-mail (read-after-write de createUser).
func (r *Resolver) loadUserByEmail(ctx context.Context, email string) (*model.User, error) {
	u, err := cqrs.Ask[uquery.GetUserByEmail, *user.User](ctx, r.Queries, uquery.GetUserByEmail{Email: email})
	if err != nil {
		return nil, err
	}
	return toGraphUser(u), nil
}

func (r *Resolver) listUsers(ctx context.Context) ([]*model.User, error) {
	users, err := cqrs.Ask[uquery.ListUsers, []*user.User](ctx, r.Queries, uquery.ListUsers{})
	if err != nil {
		return nil, err
	}
	out := make([]*model.User, 0, len(users))
	for _, u := range users {
		out = append(out, toGraphUser(u))
	}
	return out, nil
}

func toGraphUser(u *user.User) *model.User {
	return &model.User{ID: u.ID().String(), Name: u.Name().String(), Email: u.Email().String()}
}
