package adapters

import (
	"github.com/google/uuid"

	"github.com/example/payment-federation/internal/payments/application/port"
	"github.com/example/payment-federation/internal/payments/domain/order"
	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// UUIDGenerator implementa port.IDGenerator (pagamentos).
type UUIDGenerator struct{}

var _ port.IDGenerator = UUIDGenerator{}

func (UUIDGenerator) NewID() payment.PaymentID {
	return payment.PaymentID("pay_" + uuid.NewString())
}

// OrderUUIDGenerator implementa port.OrderIDGenerator (pedidos).
type OrderUUIDGenerator struct{}

var _ port.OrderIDGenerator = OrderUUIDGenerator{}

func (OrderUUIDGenerator) NewID() order.OrderID {
	return order.OrderID("ord_" + uuid.NewString())
}
