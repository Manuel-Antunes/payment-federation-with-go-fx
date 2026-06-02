package bus

import (
	"context"

	"go.uber.org/zap"

	"github.com/example/payment-federation/internal/shared/cqrs"
	"github.com/example/payment-federation/internal/user/application/event"
)

// RegisterEvents registra os consumidores de eventos do módulo de usuário via
// cqrs.RegisterEvent (que encapsula o EventProcessor do Watermill):
//   - a projeção de usuário (observabilidade / read models).
func RegisterEvents(events *cqrs.EventBus, log *zap.Logger) error {
	return cqrs.RegisterEvent(events, "user_projection",
		userProjection{log: log.Named("user_projection")})
}

// userProjection consome eventos de integração de usuário (observabilidade /
// read models).
type userProjection struct {
	log *zap.Logger
}

func (p userProjection) Handle(ctx context.Context, evt *event.UserIntegrationEvent) error {
	p.log.Info("user_event_consumed",
		zap.String("event", evt.Name),
		zap.String("user_id", evt.UserID),
		zap.Time("occurred_at", evt.OccurredAt),
	)
	return nil
}
