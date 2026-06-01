package order

import "time"

// DomainEvent é o contrato de eventos emitidos pelo agregado Order.
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
	AggregateID() OrderID
}

type baseEvent struct {
	id OrderID
	at time.Time
}

func (e baseEvent) OccurredAt() time.Time  { return e.at }
func (e baseEvent) AggregateID() OrderID   { return e.id }

// OrderCreated é emitido na criação do pedido. Carrega o que o consumidor
// (saga de pagamento) precisa para iniciar o pagamento.
type OrderCreated struct {
	baseEvent
	CustomerID  string
	AmountCents int64
	Currency    string
}

func (OrderCreated) EventName() string { return "order.created" }
