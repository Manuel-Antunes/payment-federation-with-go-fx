package order

import "context"

// Repository é a PORTA de persistência do agregado Order.
type Repository interface {
	// Save persiste um pedido novo de forma idempotente pela IdempotencyKey
	// (ErrAlreadyExists se a chave já existir).
	Save(ctx context.Context, o *Order) error
	// FindByID hidrata um pedido pelo id (ErrNotFound se não existir).
	FindByID(ctx context.Context, id OrderID) (*Order, error)
	// FindByIDs hidrata, em lote, os pedidos dos ids dados (ignora ausentes).
	// É a leitura por trás do DataLoader de pedidos.
	FindByIDs(ctx context.Context, ids []OrderID) ([]*Order, error)
	// FindByIdempotencyKey resolve o pedido pela chave (read-after-write).
	FindByIdempotencyKey(ctx context.Context, key string) (*Order, error)
}
