// Package order é o agregado Order do bounded context de pagamentos. Um pedido,
// ao ser criado, emite o evento de domínio OrderCreated — que (via barramento)
// dispara o processamento do pagamento. Reutiliza os Value Objects de dinheiro
// do pacote payment (mesmo bounded context).
package order

import (
	"time"

	"github.com/example/payment-federation/internal/payments/domain/payment"
	"github.com/example/payment-federation/internal/shared/domain"
)

// Clock é injetável para testes determinísticos.
type Clock func() time.Time

// Order é o Aggregate Root. Estado privado, mutado só por métodos que validam.
type Order struct {
	id             OrderID
	idempotencyKey *payment.IdempotencyKey
	customerID     string
	amount         payment.Money
	status         Status
	createdAt      time.Time
	version        int

	// AggregateRoot fornece Record/PullEvents (composição da base de domínio).
	domain.AggregateRoot[domain.DomainEvent]
}

// NewOrder é a fábrica do agregado. Exige um cliente e um valor válido, e emite
// OrderCreated. Recebe um id já gerado (porta de identidade fora do domínio).
func NewOrder(id OrderID, key *payment.IdempotencyKey, customerID string, amount payment.Money, now Clock) (*Order, error) {
	if id.IsZero() {
		return nil, ErrNotFound
	}
	if customerID == "" {
		return nil, ErrEmptyCustomer
	}
	if amount.IsZero() {
		return nil, payment.ErrNonPositiveAmount
	}
	t := now()
	o := &Order{
		id:             id,
		idempotencyKey: key,
		customerID:     customerID,
		amount:         amount,
		status:         StatusCreated,
		createdAt:      t,
		version:        1,
	}
	o.Apply(OrderCreated{
		BaseEvent:      domain.NewBaseEvent(id, t),
		CustomerID:     customerID,
		IdempotenceKey: o.IdempotencyKey(),
		AmountCents:    amount.AmountCents(),
		Currency:       string(amount.Currency()),
	})
	return o, nil
}

// Record/PullEvents vêm da composição com domain.AggregateRoot.

// ---- Getters --------------------------------------------------------------

func (o *Order) ID() OrderID { return o.id }
func (o *Order) IdempotencyKey() payment.IdempotencyKey {
	if o.idempotencyKey == nil {
		return payment.IdempotencyKey(o.id)
	}
	return *o.idempotencyKey
}
func (o *Order) CustomerID() string    { return o.customerID }
func (o *Order) Amount() payment.Money { return o.amount }
func (o *Order) Status() Status        { return o.status }
func (o *Order) CreatedAt() time.Time  { return o.createdAt }
func (o *Order) Version() int          { return o.version }

// ---- Snapshot (persistência) ----------------------------------------------

type Snapshot struct {
	ID             string
	IdempotencyKey *string
	CustomerID     string
	AmountCents    int64
	Currency       string
	Status         string
	CreatedAt      time.Time
	Version        int
}

func (o *Order) ToSnapshot() Snapshot {
	var idempotencyKey *string
	if o.idempotencyKey != nil {
		key := o.idempotencyKey.String()
		idempotencyKey = &key
	}
	return Snapshot{
		ID:             o.id.String(),
		IdempotencyKey: idempotencyKey,
		CustomerID:     o.customerID,
		AmountCents:    o.amount.AmountCents(),
		Currency:       string(o.amount.Currency()),
		Status:         string(o.status),
		CreatedAt:      o.createdAt,
		Version:        o.version,
	}
}

// FromSnapshot reidrata o agregado sem revalidar nem emitir eventos.
func FromSnapshot(s Snapshot) *Order {
	// O valor já foi validado quando gravado; reconstruímos direto.
	amount, _ := payment.NewMoney(s.AmountCents, payment.Currency(s.Currency))
	var idempotencyKey *payment.IdempotencyKey
	if s.IdempotencyKey != nil {
		key := payment.IdempotencyKey(*s.IdempotencyKey)
		idempotencyKey = &key
	}
	return &Order{
		id:             OrderID(s.ID),
		idempotencyKey: idempotencyKey,
		customerID:     s.CustomerID,
		amount:         amount,
		status:         Status(s.Status),
		createdAt:      s.CreatedAt,
		version:        s.Version,
	}
}
