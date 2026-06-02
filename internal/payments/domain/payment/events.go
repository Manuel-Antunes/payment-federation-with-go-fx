package payment

import "github.com/example/payment-federation/internal/shared/domain"

// Os eventos de pagamento embutem domain.BaseEvent[PaymentID] (id + instante) e
// só declaram EventName + seus campos próprios. Satisfazem domain.DomainEvent.

type PaymentInitiated struct {
	domain.BaseEvent[PaymentID]
	Amount Money
}

func (PaymentInitiated) EventName() string { return "payment.initiated" }

type PaymentAuthorized struct {
	domain.BaseEvent[PaymentID]
	GatewayRef GatewayReference
}

func (PaymentAuthorized) EventName() string { return "payment.authorized" }

type PaymentCaptured struct {
	domain.BaseEvent[PaymentID]
	Amount Money
}

func (PaymentCaptured) EventName() string { return "payment.captured" }

type PaymentRefunded struct {
	domain.BaseEvent[PaymentID]
	Amount  Money
	Partial bool
}

func (PaymentRefunded) EventName() string { return "payment.refunded" }

type PaymentFailed struct {
	domain.BaseEvent[PaymentID]
	Reason string
}

func (PaymentFailed) EventName() string { return "payment.failed" }
