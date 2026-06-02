package order

import "github.com/example/payment-federation/internal/shared/domain"

// OrderCreated embute domain.BaseEvent[OrderID] (id + instante) e só declara
// EventName + seus campos próprios. Satisfaz domain.DomainEvent.
//
// OrderCreated é emitido na criação do pedido. Carrega o que o consumidor
// (saga de pagamento) precisa para iniciar o pagamento.
type OrderCreated struct {
	domain.BaseEvent[OrderID]
	CustomerID  string
	AmountCents int64
	Currency    string
}

func (OrderCreated) EventName() string { return "order.created" }
