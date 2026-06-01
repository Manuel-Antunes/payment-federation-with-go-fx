// Package order é o agregado Order do bounded context de pagamentos. Um pedido,
// ao ser criado, emite o evento de domínio OrderCreated — que (via barramento)
// dispara o processamento do pagamento. Reutiliza os Value Objects de dinheiro
// do pacote payment (mesmo bounded context).
package order

import (
	"time"

	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// Clock é injetável para testes determinísticos.
type Clock func() time.Time

// Order é o Aggregate Root. Estado privado, mutado só por métodos que validam.
type Order struct {
	id             OrderID
	idempotencyKey payment.IdempotencyKey
	customerID     string
	amount         payment.Money
	status         Status
	createdAt      time.Time
	version        int

	events []DomainEvent
}

// NewOrder é a fábrica do agregado. Exige um cliente e um valor válido, e emite
// OrderCreated. Recebe um id já gerado (porta de identidade fora do domínio).
func NewOrder(id OrderID, key payment.IdempotencyKey, customerID string, amount payment.Money, now Clock) (*Order, error) {
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
	o.record(OrderCreated{
		baseEvent:   baseEvent{id: id, at: t},
		CustomerID:  customerID,
		AmountCents: amount.AmountCents(),
		Currency:    string(amount.Currency()),
	})
	return o, nil
}

func (o *Order) record(e DomainEvent) { o.events = append(o.events, e) }

// PullEvents devolve e limpa os eventos pendentes (após persistir com sucesso).
func (o *Order) PullEvents() []DomainEvent {
	out := o.events
	o.events = nil
	return out
}

// ---- Getters --------------------------------------------------------------

func (o *Order) ID() OrderID                          { return o.id }
func (o *Order) IdempotencyKey() payment.IdempotencyKey { return o.idempotencyKey }
func (o *Order) CustomerID() string                   { return o.customerID }
func (o *Order) Amount() payment.Money                { return o.amount }
func (o *Order) Status() Status                       { return o.status }
func (o *Order) CreatedAt() time.Time                 { return o.createdAt }
func (o *Order) Version() int                         { return o.version }

// ---- Snapshot (persistência) ----------------------------------------------

type Snapshot struct {
	ID             string
	IdempotencyKey string
	CustomerID     string
	AmountCents    int64
	Currency       string
	Status         string
	CreatedAt      time.Time
	Version        int
}

func (o *Order) ToSnapshot() Snapshot {
	return Snapshot{
		ID:             o.id.String(),
		IdempotencyKey: o.idempotencyKey.String(),
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
	return &Order{
		id:             OrderID(s.ID),
		idempotencyKey: payment.IdempotencyKey(s.IdempotencyKey),
		customerID:     s.CustomerID,
		amount:         amount,
		status:         Status(s.Status),
		createdAt:      s.CreatedAt,
		version:        s.Version,
	}
}
