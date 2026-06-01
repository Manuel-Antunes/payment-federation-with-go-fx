package adapters

import (
	"github.com/google/uuid"

	"github.com/example/payment-federation/internal/user/application/port"
	"github.com/example/payment-federation/internal/user/domain/user"
)

// UUIDGenerator implementa port.IDGenerator para o módulo de usuário.
type UUIDGenerator struct{}

var _ port.IDGenerator = UUIDGenerator{}

func (UUIDGenerator) NewID() user.UserID {
	return user.UserID("usr_" + uuid.NewString())
}
