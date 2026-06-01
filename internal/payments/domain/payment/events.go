package payment

import "time"

// DomainEvent é o contrato de eventos emitidos pelo agregado. A aplicação pode
// publicá-los (outbox, broker) DEPOIS de persistir — nunca antes.
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
	AggregateID() PaymentID
}

type baseEvent struct {
	id  PaymentID
	at  time.Time
}

func (e baseEvent) OccurredAt() time.Time   { return e.at }
func (e baseEvent) AggregateID() PaymentID  { return e.id }

type PaymentInitiated struct {
	baseEvent
	Amount Money
}

func (PaymentInitiated) EventName() string { return "payment.initiated" }

type PaymentAuthorized struct {
	baseEvent
	GatewayRef GatewayReference
}

func (PaymentAuthorized) EventName() string { return "payment.authorized" }

type PaymentCaptured struct {
	baseEvent
	Amount Money
}

func (PaymentCaptured) EventName() string { return "payment.captured" }

type PaymentRefunded struct {
	baseEvent
	Amount  Money
	Partial bool
}

func (PaymentRefunded) EventName() string { return "payment.refunded" }

type PaymentFailed struct {
	baseEvent
	Reason string
}

func (PaymentFailed) EventName() string { return "payment.failed" }
