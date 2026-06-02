// Package event define os EVENTOS DE INTEGRAÇÃO publicados pelo módulo de
// usuário no barramento — o contrato estável e serializável que cruza a
// fronteira do domínio. As funções de mapeamento traduzem os eventos de domínio
// para estes DTOs planos.
package event

import (
	"time"

	"github.com/example/payment-federation/internal/user/domain/user"
)

// UserIntegrationEvent é o evento de integração de usuário.
type UserIntegrationEvent struct {
	Name       string    `json:"name"`       // ex.: "user.created"
	UserID     string    `json:"userId"`     // id do agregado afetado
	UserName   string    `json:"userName"`   // nome de exibição
	Email      string    `json:"email"`      // e-mail
	OccurredAt time.Time `json:"occurredAt"` // instante (UTC) do evento no domínio
}

// UserIntegrationFrom traduz um UserCreated de domínio para o DTO.
func UserIntegrationFrom(e user.UserCreated) UserIntegrationEvent {
	return UserIntegrationEvent{
		Name:       e.EventName(),
		UserID:     e.AggregateID(),
		UserName:   e.Name,
		Email:      e.Email,
		OccurredAt: e.OccurredAt().UTC(),
	}
}
