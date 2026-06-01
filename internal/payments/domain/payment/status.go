package payment

// Status representa o ciclo de vida do pagamento.
//
//	PENDING ─► AUTHORIZED ─► CAPTURED ─► (PARTIALLY_)REFUNDED
//	   └────────► FAILED
type Status string

const (
	StatusPending          Status = "PENDING"
	StatusAuthorized       Status = "AUTHORIZED"
	StatusCaptured         Status = "CAPTURED"
	StatusPartiallyRefund  Status = "PARTIALLY_REFUNDED"
	StatusRefunded         Status = "REFUNDED"
	StatusFailed           Status = "FAILED"
)

// allowedTransitions codifica a máquina de estados como invariante de domínio.
var allowedTransitions = map[Status]map[Status]bool{
	StatusPending: {
		StatusAuthorized: true,
		StatusFailed:     true,
	},
	StatusAuthorized: {
		StatusCaptured: true,
		StatusFailed:   true,
	},
	StatusCaptured: {
		StatusPartiallyRefund: true,
		StatusRefunded:        true,
	},
	StatusPartiallyRefund: {
		StatusPartiallyRefund: true,
		StatusRefunded:        true,
	},
}

func (s Status) canTransitionTo(next Status) bool {
	return allowedTransitions[s][next]
}

func (s Status) IsTerminal() bool {
	return s == StatusRefunded || s == StatusFailed
}
