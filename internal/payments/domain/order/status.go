package order

// Status representa o ciclo de vida do pedido. Mínimo para o exemplo: o pedido
// nasce CREATED e o pagamento é processado de forma reativa (saga).
type Status string

const (
	StatusCreated Status = "CREATED"
)
