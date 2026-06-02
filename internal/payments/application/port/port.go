// Package port define as PORTAS (interfaces) do módulo de pagamentos que a
// infraestrutura implementa. Fica num subpacote próprio para que os use-cases
// (application/command) possam depender das portas sem criar ciclo com o
// módulo fx (package application).
package port

import (
	"context"

	"github.com/example/payment-federation/internal/payments/domain/order"
	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// IDGenerator gera identidade de pagamento.
type IDGenerator interface {
	NewID() payment.PaymentID
}

// OrderIDGenerator gera identidade de pedido.
type OrderIDGenerator interface {
	NewID() order.OrderID
}

// CustomerDirectory valida que um pedido referencia um cliente existente, sem
// acoplar o módulo de pagamentos ao de usuário (a impl é um bridge na composição).
type CustomerDirectory interface {
	Exists(ctx context.Context, customerID string) (bool, error)
}
