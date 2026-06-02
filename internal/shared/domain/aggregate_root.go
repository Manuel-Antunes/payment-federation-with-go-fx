package domain

// AggregateRoot é a base, por COMPOSIÇÃO, dos agregados (Payment, Order, User):
// acumula os eventos de domínio emitidos e os entrega (esvaziando) no PullEvents
// — chamado pela aplicação após persistir. E é o tipo de evento do agregado
// (tipicamente domain.DomainEvent).
type AggregateRoot[E any] struct {
	events []E
}

// Apply registra um ou mais eventos de domínio pendentes.
func (a *AggregateRoot[E]) Apply(events ...E) {
	a.events = append(a.events, events...)
}

// PullEvents devolve e limpa os eventos pendentes.
func (a *AggregateRoot[E]) PullEvents() []E {
	out := a.events
	a.events = nil
	return out
}
