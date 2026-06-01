package payment

import "strings"

// ---------------------------------------------------------------------------
// Value Objects
//
// VOs são imutáveis e auto-validáveis. Toda regra de "o que é um valor válido"
// vive aqui, nunca no banco e nunca no resolver GraphQL.
// ---------------------------------------------------------------------------

// PaymentID identifica unicamente um agregado Payment.
type PaymentID string

func (id PaymentID) String() string { return string(id) }
func (id PaymentID) IsZero() bool   { return id == "" }

// IdempotencyKey é a chave fornecida pelo cliente que garante que a mesma
// requisição de pagamento nunca seja processada duas vezes.
type IdempotencyKey string

func NewIdempotencyKey(raw string) (IdempotencyKey, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 8 {
		return "", ErrInvalidIdempotencyKey
	}
	return IdempotencyKey(raw), nil
}

func (k IdempotencyKey) String() string { return string(k) }

// Currency é um VO restrito a códigos ISO-4217 suportados.
type Currency string

const (
	BRL Currency = "BRL"
	USD Currency = "USD"
	EUR Currency = "EUR"
)

var supportedCurrencies = map[Currency]bool{BRL: true, USD: true, EUR: true}

func NewCurrency(raw string) (Currency, error) {
	c := Currency(strings.ToUpper(strings.TrimSpace(raw)))
	if !supportedCurrencies[c] {
		return "", ErrUnsupportedCurrency
	}
	return c, nil
}

// Money guarda valores em centavos (inteiro) para evitar erros de ponto
// flutuante. É um VO: operações retornam novas instâncias.
type Money struct {
	amountCents int64
	currency    Currency
}

func NewMoney(amountCents int64, currency Currency) (Money, error) {
	if amountCents <= 0 {
		return Money{}, ErrNonPositiveAmount
	}
	if !supportedCurrencies[currency] {
		return Money{}, ErrUnsupportedCurrency
	}
	return Money{amountCents: amountCents, currency: currency}, nil
}

func (m Money) AmountCents() int64 { return m.amountCents }
func (m Money) Currency() Currency { return m.currency }
func (m Money) IsZero() bool       { return m.amountCents == 0 }

// SameCurrency garante que operações só ocorram entre moedas iguais.
func (m Money) SameCurrency(other Money) error {
	if m.currency != other.currency {
		return ErrCurrencyMismatch
	}
	return nil
}

// Subtract retorna m - other, validando moeda e não permitindo negativo.
func (m Money) Subtract(other Money) (Money, error) {
	if err := m.SameCurrency(other); err != nil {
		return Money{}, err
	}
	if other.amountCents > m.amountCents {
		return Money{}, ErrInsufficientAmount
	}
	return Money{amountCents: m.amountCents - other.amountCents, currency: m.currency}, nil
}

// GatewayReference é o identificador devolvido pelo gateway externo
// (ex.: charge id da Stripe / id da transação Adyen).
type GatewayReference string

func (r GatewayReference) String() string { return string(r) }
func (r GatewayReference) IsEmpty() bool  { return r == "" }
