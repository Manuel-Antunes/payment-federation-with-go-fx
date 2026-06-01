// Package persistence implementa as portas de repositório do módulo de
// pagamentos sobre o ent (Postgres). O mapeamento ent <-> domínio fica nos
// arquivos *_schema.go.
package persistence

import (
	"context"

	"github.com/example/payment-federation/internal/payments/domain/payment"
	entx "github.com/example/payment-federation/internal/payments/infrastructure/ent"
	entpayment "github.com/example/payment-federation/internal/payments/infrastructure/ent/payment"
)

// PaymentEntRepository é o adaptador ent da porta payment.Repository.
type PaymentEntRepository struct {
	client *entx.Client
	mapper paymentSchema
}

func NewPaymentEntRepository(client *entx.Client) *PaymentEntRepository {
	return &PaymentEntRepository{client: client}
}

var _ payment.Repository = (*PaymentEntRepository)(nil)

func (r *PaymentEntRepository) Save(ctx context.Context, p *payment.Payment) error {
	_, err := r.mapper.toCreate(r.client, p).Save(ctx)
	if entx.IsConstraintError(err) {
		return payment.ErrAlreadyExists // idempotency_key único violado
	}
	return err
}

// Update aplica concorrência otimista: só atualiza se a versão no banco ainda
// for a anterior (s.Version-1). 0 linhas afetadas => conflito/ausente.
func (r *PaymentEntRepository) Update(ctx context.Context, p *payment.Payment) error {
	s := p.ToSnapshot()
	affected, err := r.client.Payment.Update().
		Where(entpayment.IDEQ(s.ID), entpayment.VersionEQ(s.Version-1)).
		SetAmountCents(s.AmountCents).
		SetRefundedCents(s.RefundedCents).
		SetStatus(s.Status).
		SetGatewayRef(s.GatewayRef).
		SetUpdatedAt(s.UpdatedAt).
		SetVersion(s.Version).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return payment.ErrInvalidTransition // versão divergente ou não encontrado
	}
	return nil
}

func (r *PaymentEntRepository) FindByID(ctx context.Context, id payment.PaymentID) (*payment.Payment, error) {
	row, err := r.client.Payment.Get(ctx, id.String())
	if entx.IsNotFound(err) {
		return nil, payment.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.mapper.toDomain(row), nil
}

func (r *PaymentEntRepository) FindByIdempotencyKey(ctx context.Context, key payment.IdempotencyKey) (*payment.Payment, error) {
	row, err := r.client.Payment.Query().Where(entpayment.IdempotencyKey(key.String())).Only(ctx)
	if entx.IsNotFound(err) {
		return nil, payment.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.mapper.toDomain(row), nil
}
