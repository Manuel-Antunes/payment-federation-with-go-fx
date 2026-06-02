package cqrs

import (
	"context"

	"github.com/example/payment-federation/internal/shared/domain"
)

// EventSource é qualquer agregado capaz de entregar seus eventos de domínio
// pendentes (satisfeito por quem embute domain.AggregateRoot[domain.DomainEvent]).
type EventSource interface {
	PullEvents() []domain.DomainEvent
}

// IntegrationMapper traduz um evento de DOMÍNIO no evento de INTEGRAÇÃO (DTO
// serializável) a publicar no barramento. Retorna ok=false para pular o evento.
type IntegrationMapper func(domain.DomainEvent) (any, bool)

// EventPublisher é a PORTA de publicação GENÉRICA de eventos de domínio: dado um
// agregado, devolve um Outbox que comita seus eventos pendentes no barramento.
// É um contrato único (não um publisher por agregado) — a parte específica de
// cada agregado é só o IntegrationMapper passado no Commit. Os command handlers
// dependem desta interface; a impl concreta é injetada pelo fx.
type EventPublisher interface {
	// For captura os eventos pendentes do agregado e devolve um Outbox pronto
	// para commitar — o "publisher exato" para os eventos daquele agregado.
	For(agg EventSource) *Outbox
}

// eventPublisher é a implementação da porta sobre o nosso EventBus (que, por sua
// vez, envelopa o Watermill).
type eventPublisher struct {
	bus *EventBus
}

func NewEventPublisher(bus *EventBus) EventPublisher {
	return &eventPublisher{bus: bus}
}

func (p *eventPublisher) For(agg EventSource) *Outbox {
	return &Outbox{bus: p.bus, events: agg.PullEvents()}
}

// Outbox guarda os eventos puxados de um agregado até o Commit.
type Outbox struct {
	bus    *EventBus
	events []domain.DomainEvent
}

// Commit mapeia cada evento de domínio para o evento de integração e o publica.
// Falha no primeiro erro de publicação.
func (o *Outbox) Commit(ctx context.Context, mapper IntegrationMapper) error {
	for _, e := range o.events {
		integration, ok := mapper(e)
		if !ok {
			continue
		}
		if err := o.bus.Publish(ctx, integration); err != nil {
			return err
		}
	}
	return nil
}
