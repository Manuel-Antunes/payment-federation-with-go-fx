package user

import "github.com/example/payment-federation/internal/shared/domain"

// UserCreated é emitido na criação do usuário. Embute domain.BaseEvent[UserID]
// (id + instante) e só declara EventName + seus campos próprios. Satisfaz
// domain.DomainEvent.
type UserCreated struct {
	domain.BaseEvent[UserID]
	Name  string
	Email string
}

func (UserCreated) EventName() string { return "user.created" }
