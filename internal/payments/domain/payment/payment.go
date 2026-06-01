package payment

import "time"

// ---------------------------------------------------------------------------
// Aggregate Root: Payment
//
// Toda a regra de negócio (invariantes, transições, limites de reembolso)
// vive aqui. Os campos são privados: o estado só muda por métodos de comando
// do agregado, que validam ANTES de mutar. Nenhuma camada externa consegue
// colocar o agregado num estado inválido — e o repositório só persiste o que
// o domínio já considerou válido.
// ---------------------------------------------------------------------------

type Payment struct {
	id             PaymentID
	idempotencyKey IdempotencyKey
	customerID     string
	amount         Money // valor autorizado/cobrado
	refunded       Money // total já reembolsado
	status         Status
	gatewayRef     GatewayReference
	createdAt      time.Time
	updatedAt      time.Time
	version        int // controle de concorrência otimista

	events []DomainEvent
}

// Clock é injetável para testes determinísticos.
type Clock func() time.Time

// NewPayment é a fábrica do agregado. Aplica todas as validações de criação.
// Recebe um id já gerado (porta de identidade fica fora do domínio puro).
func NewPayment(id PaymentID, key IdempotencyKey, customerID string, amount Money, now Clock) (*Payment, error) {
	if id.IsZero() {
		return nil, ErrNotFound
	}
	if customerID == "" {
		return nil, ErrEmptyCustomer
	}
	if amount.IsZero() {
		return nil, ErrNonPositiveAmount
	}
	t := now()
	p := &Payment{
		id:             id,
		idempotencyKey: key,
		customerID:     customerID,
		amount:         amount,
		refunded:       Money{currency: amount.Currency()}, // 0 cents
		status:         StatusPending,
		createdAt:      t,
		updatedAt:      t,
		version:        1,
	}
	p.record(PaymentInitiated{baseEvent: baseEvent{id: id, at: t}, Amount: amount})
	return p, nil
}

// Authorize transiciona PENDING -> AUTHORIZED após sucesso no gateway.
// A referência do gateway é parte do resultado de negócio.
func (p *Payment) Authorize(ref GatewayReference, now Clock) error {
	if p.status == StatusAuthorized {
		return ErrAlreadyAuthorized
	}
	if !p.status.canTransitionTo(StatusAuthorized) {
		return ErrInvalidTransition
	}
	if ref.IsEmpty() {
		return ErrNotAuthorized
	}
	t := now()
	p.gatewayRef = ref
	p.transition(StatusAuthorized, t)
	p.record(PaymentAuthorized{baseEvent: baseEvent{id: p.id, at: t}, GatewayRef: ref})
	return nil
}

// Capture transiciona AUTHORIZED -> CAPTURED.
func (p *Payment) Capture(now Clock) error {
	if p.status == StatusCaptured {
		return ErrAlreadyCaptured
	}
	if p.status != StatusAuthorized {
		return ErrNotAuthorized
	}
	if !p.status.canTransitionTo(StatusCaptured) {
		return ErrInvalidTransition
	}
	t := now()
	p.transition(StatusCaptured, t)
	p.record(PaymentCaptured{baseEvent: baseEvent{id: p.id, at: t}, Amount: p.amount})
	return nil
}

// Refund aplica um reembolso (total ou parcial). Regra central: a soma dos
// reembolsos nunca pode exceder o valor capturado.
func (p *Payment) Refund(amount Money, now Clock) error {
	if p.status != StatusCaptured && p.status != StatusPartiallyRefund {
		return ErrInvalidTransition
	}
	if err := p.amount.SameCurrency(amount); err != nil {
		return err
	}
	newRefunded := Money{amountCents: p.refunded.amountCents + amount.amountCents, currency: p.amount.currency}
	if newRefunded.amountCents > p.amount.amountCents {
		return ErrRefundExceedsTotal
	}

	t := now()
	p.refunded = newRefunded
	partial := newRefunded.amountCents < p.amount.amountCents
	next := StatusRefunded
	if partial {
		next = StatusPartiallyRefund
	}
	if !p.status.canTransitionTo(next) {
		return ErrInvalidTransition
	}
	p.transition(next, t)
	p.record(PaymentRefunded{baseEvent: baseEvent{id: p.id, at: t}, Amount: amount, Partial: partial})
	return nil
}

// Fail marca o pagamento como falho (ex.: gateway recusou).
func (p *Payment) Fail(reason string, now Clock) error {
	if !p.status.canTransitionTo(StatusFailed) {
		return ErrInvalidTransition
	}
	t := now()
	p.transition(StatusFailed, t)
	p.record(PaymentFailed{baseEvent: baseEvent{id: p.id, at: t}, Reason: reason})
	return nil
}

func (p *Payment) transition(next Status, t time.Time) {
	p.status = next
	p.updatedAt = t
	p.version++
}

func (p *Payment) record(e DomainEvent) { p.events = append(p.events, e) }

// PullEvents devolve e limpa os eventos pendentes (chamado pela aplicação
// após persistir com sucesso).
func (p *Payment) PullEvents() []DomainEvent {
	out := p.events
	p.events = nil
	return out
}

// ---- Getters (read-only) ---------------------------------------------------

func (p *Payment) ID() PaymentID                { return p.id }
func (p *Payment) IdempotencyKey() IdempotencyKey { return p.idempotencyKey }
func (p *Payment) CustomerID() string           { return p.customerID }
func (p *Payment) Amount() Money                 { return p.amount }
func (p *Payment) Refunded() Money               { return p.refunded }
func (p *Payment) Status() Status                { return p.status }
func (p *Payment) GatewayRef() GatewayReference  { return p.gatewayRef }
func (p *Payment) CreatedAt() time.Time          { return p.createdAt }
func (p *Payment) UpdatedAt() time.Time          { return p.updatedAt }
func (p *Payment) Version() int                  { return p.version }

// ---- Reconstrução a partir da persistência (hidratação) --------------------

// Snapshot é a representação plana usada pelo repositório para gravar/ler.
// Mantém o agregado desacoplado do schema de banco.
type Snapshot struct {
	ID             string
	IdempotencyKey string
	CustomerID     string
	AmountCents    int64
	Currency       string
	RefundedCents  int64
	Status         string
	GatewayRef     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Version        int
}

// ToSnapshot serializa o estado atual (sem eventos).
func (p *Payment) ToSnapshot() Snapshot {
	return Snapshot{
		ID:             p.id.String(),
		IdempotencyKey: p.idempotencyKey.String(),
		CustomerID:     p.customerID,
		AmountCents:    p.amount.AmountCents(),
		Currency:       string(p.amount.Currency()),
		RefundedCents:  p.refunded.amountCents,
		Status:         string(p.status),
		GatewayRef:     p.gatewayRef.String(),
		CreatedAt:      p.createdAt,
		UpdatedAt:      p.updatedAt,
		Version:        p.version,
	}
}

// FromSnapshot reidrata o agregado SEM disparar eventos nem validações de
// criação (o estado já foi válido quando gravado).
func FromSnapshot(s Snapshot) *Payment {
	cur := Currency(s.Currency)
	return &Payment{
		id:             PaymentID(s.ID),
		idempotencyKey: IdempotencyKey(s.IdempotencyKey),
		customerID:     s.CustomerID,
		amount:         Money{amountCents: s.AmountCents, currency: cur},
		refunded:       Money{amountCents: s.RefundedCents, currency: cur},
		status:         Status(s.Status),
		gatewayRef:     GatewayReference(s.GatewayRef),
		createdAt:      s.CreatedAt,
		updatedAt:      s.UpdatedAt,
		version:        s.Version,
	}
}
