// Package port define as PORTAS do módulo de usuário (num subpacote próprio
// para evitar ciclo entre os use-cases e o módulo fx).
package port

import "github.com/example/payment-federation/internal/user/domain/user"

// IDGenerator gera identidade de usuário.
type IDGenerator interface {
	NewID() user.UserID
}
