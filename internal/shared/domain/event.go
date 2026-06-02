package domain

import "time"

// DomainEvent é o contrato de TODO evento de domínio. AggregateID devolve o id
// do agregado como string (uniforme para mapeamento/serialização na borda).
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
	AggregateID() string
}

// BaseEvent é embutido pelos eventos concretos; ID é o tipo de id do agregado
// (PaymentID, OrderID, UserID...) aplicado como type parameter. Fornece
// OccurredAt/AggregateID, então cada evento só precisa declarar EventName +
// seus campos próprios.
type BaseEvent[ID ~string] struct {
	id ID
	at time.Time
}

// NewBaseEvent cria a base de um evento (id do agregado + instante de ocorrência).
func NewBaseEvent[ID ~string](id ID, at time.Time) BaseEvent[ID] {
	return BaseEvent[ID]{id: id, at: at}
}

func (e BaseEvent[ID]) OccurredAt() time.Time { return e.at }
func (e BaseEvent[ID]) AggregateID() string   { return string(e.id) }
