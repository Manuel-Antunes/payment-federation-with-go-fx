package payment

import "errors"

// Erros de domínio. São puros (sem dependência de infra) para que qualquer
// camada possa traduzi-los — HTTP, GraphQL, gRPC etc.
var (
	ErrInvalidIdempotencyKey = errors.New("payment: chave de idempotência inválida (mínimo 8 caracteres)")
	ErrUnsupportedCurrency   = errors.New("payment: moeda não suportada")
	ErrNonPositiveAmount     = errors.New("payment: valor deve ser maior que zero")
	ErrCurrencyMismatch      = errors.New("payment: moedas divergentes")
	ErrInsufficientAmount    = errors.New("payment: valor insuficiente")

	// Invariantes / transições de estado
	ErrInvalidTransition  = errors.New("payment: transição de estado inválida")
	ErrAlreadyAuthorized  = errors.New("payment: pagamento já autorizado")
	ErrNotAuthorized      = errors.New("payment: pagamento não está autorizado")
	ErrAlreadyCaptured    = errors.New("payment: pagamento já capturado")
	ErrRefundExceedsTotal = errors.New("payment: reembolso excede o valor capturado")
	ErrEmptyCustomer      = errors.New("payment: cliente é obrigatório")

	// Persistência / idempotência (porta)
	ErrNotFound      = errors.New("payment: não encontrado")
	ErrAlreadyExists = errors.New("payment: já existe um pagamento para esta chave de idempotência")
)
