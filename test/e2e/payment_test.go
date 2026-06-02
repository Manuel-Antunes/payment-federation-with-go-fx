package e2e

import "testing"

func TestPaymentLifecycle(t *testing.T) {
	e := newE2E(t)

	p := e.processPayment(map[string]any{
		"idempotencyKey": "pay-lifecycle-1",
		"customerId":     "cust_123",
		"amountCents":    5000,
		"currency":       "BRL",
	})
	if p.Status != "CAPTURED" || p.AmountCents != 5000 {
		t.Fatalf("unexpected payment after process: %+v", p)
	}

	// read-after-write por id
	var q struct {
		Payment paymentView `json:"payment"`
	}
	e.mustQuery(`query($id: ID!){ payment(id:$id){ id status refundedCents } }`,
		map[string]any{"id": p.ID}, &q)
	if q.Payment.ID != p.ID || q.Payment.Status != "CAPTURED" {
		t.Fatalf("unexpected payment on query: %+v", q.Payment)
	}

	// refund parcial -> PARTIALLY_REFUNDED
	var r1 struct {
		RefundPayment paymentView `json:"refundPayment"`
	}
	e.mustQuery(`mutation($in: RefundPaymentInput!){ refundPayment(input:$in){ status refundedCents } }`,
		map[string]any{"in": map[string]any{"paymentId": p.ID, "amountCents": 2000}}, &r1)
	if r1.RefundPayment.Status != "PARTIALLY_REFUNDED" || r1.RefundPayment.RefundedCents != 2000 {
		t.Fatalf("unexpected partial refund: %+v", r1.RefundPayment)
	}

	// refund do restante -> REFUNDED
	var r2 struct {
		RefundPayment paymentView `json:"refundPayment"`
	}
	e.mustQuery(`mutation($in: RefundPaymentInput!){ refundPayment(input:$in){ status refundedCents } }`,
		map[string]any{"in": map[string]any{"paymentId": p.ID, "amountCents": 3000}}, &r2)
	if r2.RefundPayment.Status != "REFUNDED" || r2.RefundPayment.RefundedCents != 5000 {
		t.Fatalf("unexpected full refund: %+v", r2.RefundPayment)
	}
}

func TestPaymentIdempotency(t *testing.T) {
	e := newE2E(t)

	in := map[string]any{
		"idempotencyKey": "pay-idem-1",
		"customerId":     "cust_9",
		"amountCents":    1500,
		"currency":       "BRL",
	}
	first := e.processPayment(in)
	second := e.processPayment(in) // mesma chave => mesmo pagamento, sem nova cobrança
	if first.ID != second.ID {
		t.Fatalf("idempotency broken: %s != %s", first.ID, second.ID)
	}
}

func TestPaymentInvalidInput(t *testing.T) {
	e := newE2E(t)

	if msg := e.mustError(`mutation($in: ProcessPaymentInput!){ processPayment(input:$in){ id } }`,
		map[string]any{"in": map[string]any{
			"idempotencyKey": "pay-bad-cur",
			"customerId":     "cust_1",
			"amountCents":    100,
			"currency":       "XYZ", // moeda não suportada
		}}); msg == "" {
		t.Fatal("expected error for unsupported currency")
	}

	// payment de id inexistente => erro
	if msg := e.mustError(`query($id: ID!){ payment(id:$id){ id } }`,
		map[string]any{"id": "pay_nope"}); msg == "" {
		t.Fatal("expected error for unknown payment id")
	}
}
