package model

// Models escritos à mão (autobind no gqlgen.yml). Mantê-los aqui evita que o
// gqlgen os regenere e nos dá controle do mapeamento de tipos.

type Payment struct {
	ID            string  `json:"id"`
	CustomerID    string  `json:"customerId"`
	AmountCents   int64   `json:"amountCents"`
	Currency      string  `json:"currency"`
	RefundedCents int64   `json:"refundedCents"`
	Status        string  `json:"status"`
	GatewayRef    *string `json:"gatewayRef,omitempty"`
}

// IsEntity marca Payment como entity da federação (exigido pelo plugin).
func (Payment) IsEntity() {}

type ProcessPaymentInput struct {
	IdempotencyKey string  `json:"idempotencyKey"`
	CustomerID     string  `json:"customerId"`
	AmountCents    int64   `json:"amountCents"`
	Currency       string  `json:"currency"`
	GatewayName    *string `json:"gatewayName,omitempty"`
}

type RefundPaymentInput struct {
	PaymentID   string `json:"paymentId"`
	AmountCents int64  `json:"amountCents"`
}

type Order struct {
	ID          string `json:"id"`
	CustomerID  string `json:"customerId"`
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
}

// IsEntity marca Order como entity da federação.
func (Order) IsEntity() {}

type CreateOrderInput struct {
	IdempotencyKey string `json:"idempotencyKey"`
	CustomerID     string `json:"customerId"`
	AmountCents    int64  `json:"amountCents"`
	Currency       string `json:"currency"`
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// IsEntity marca User como entity da federação.
func (User) IsEntity() {}

type CreateUserInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateUserInput struct {
	ID    string  `json:"id"`
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}
