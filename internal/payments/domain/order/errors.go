package order

import "errors"

var (
	ErrEmptyCustomer   = errors.New("order: cliente é obrigatório")
	ErrCustomerUnknown = errors.New("order: cliente não encontrado")

	// Persistência (porta)
	ErrNotFound      = errors.New("order: não encontrado")
	ErrAlreadyExists = errors.New("order: já existe um pedido para esta chave de idempotência")
)
