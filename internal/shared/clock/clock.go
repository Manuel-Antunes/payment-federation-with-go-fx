// Package clock é um pequeno kernel COMPARTILHADO entre módulos: injeta tempo
// de forma testável. Não conhece nada de pagamentos, então pode ser reutilizado
// por qualquer bounded context (payments, billing, etc.).
package clock

import "time"

// Clock injeta o tempo de forma testável.
type Clock interface {
	Now() time.Time
}

// SystemClock é a implementação padrão (relógio do sistema, em UTC).
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
