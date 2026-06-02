package e2e

import (
	"strings"
	"testing"
)

// TestInputValidationDirective verifica que a diretiva @binding rejeita inputs
// inválidos NA BORDA (antes de despachar o command), com mensagem clara
// referente ao campo — e que inputs válidos passam.
func TestInputValidationDirective(t *testing.T) {
	e := newE2E(t)

	cases := []struct {
		name    string
		op      string
		vars    map[string]any
		wantMsg string // substring esperado na mensagem de erro
	}{
		{
			name: "idempotencyKey curta (minLength)",
			op:   `mutation($in: ProcessPaymentInput!){ processPayment(input:$in){ id } }`,
			vars: map[string]any{"in": map[string]any{
				"idempotencyKey": "short", "customerId": "c1", "amountCents": 100, "currency": "BRL"}},
			wantMsg: "mínimo 8",
		},
		{
			name: "amountCents <= 0 (min)",
			op:   `mutation($in: ProcessPaymentInput!){ processPayment(input:$in){ id } }`,
			vars: map[string]any{"in": map[string]any{
				"idempotencyKey": "valid-key-1", "customerId": "c1", "amountCents": 0, "currency": "BRL"}},
			wantMsg: ">= 1",
		},
		{
			name: "currency fora do conjunto (oneOf)",
			op:   `mutation($in: ProcessPaymentInput!){ processPayment(input:$in){ id } }`,
			vars: map[string]any{"in": map[string]any{
				"idempotencyKey": "valid-key-1", "customerId": "c1", "amountCents": 100, "currency": "GBP"}},
			wantMsg: "deve ser um de",
		},
		{
			name: "email inválido (format)",
			op:   `mutation($in: CreateUserInput!){ createUser(input:$in){ id } }`,
			vars: map[string]any{"in": map[string]any{"name": "X", "email": "nope"}},
			wantMsg: "e-mail inválido",
		},
		{
			name: "nome em branco (notBlank)",
			op:   `mutation($in: CreateUserInput!){ createUser(input:$in){ id } }`,
			vars: map[string]any{"in": map[string]any{"name": "   ", "email": "ok@example.com"}},
			wantMsg: "não pode ser vazio",
		},
		{
			name: "createOrder amount inválido (min)",
			op:   `mutation($in: CreateOrderInput!){ createOrder(input:$in){ id } }`,
			vars: map[string]any{"in": map[string]any{
				"idempotencyKey": "ord-key-123", "customerId": "usr_x", "amountCents": -5, "currency": "BRL"}},
			wantMsg: ">= 1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := e.mustError(tc.op, tc.vars)
			if !strings.Contains(msg, tc.wantMsg) {
				t.Fatalf("expected error containing %q, got %q", tc.wantMsg, msg)
			}
		})
	}

	// input válido continua passando pela diretiva.
	p := e.processPayment(map[string]any{
		"idempotencyKey": "valid-after-dir",
		"customerId":     "c1",
		"amountCents":    100,
		"currency":       "USD",
	})
	if p.Status != "CAPTURED" {
		t.Fatalf("valid input should pass the directive, got status %q", p.Status)
	}
}
