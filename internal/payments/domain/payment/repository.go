package payment

import "context"

// Repository é uma PORTA (interface) definida pelo domínio e implementada pela
// infraestrutura. O domínio nunca conhece SQL, Mongo, etc.
//
// A idempotência de persistência é responsabilidade desta porta:
//   - Save deve falhar com ErrAlreadyExists se a IdempotencyKey já existir.
type Repository interface {
	// Save persiste um agregado novo de forma idempotente pela IdempotencyKey.
	Save(ctx context.Context, p *Payment) error

	// Update grava mutações de um agregado existente respeitando a versão
	// (concorrência otimista).
	Update(ctx context.Context, p *Payment) error

	// FindByID hidrata um agregado pelo seu id.
	FindByID(ctx context.Context, id PaymentID) (*Payment, error)

	// FindByIdempotencyKey devolve o pagamento já criado para a chave, se houver.
	// Retorna (nil, ErrNotFound) quando não existe.
	FindByIdempotencyKey(ctx context.Context, key IdempotencyKey) (*Payment, error)
}
