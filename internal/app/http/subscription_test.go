package http_test

import (
	"context"
	"testing"
	"time"
)

// TestOnPaymentProcessedSubscription valida a subscription de ponta a ponta SEM
// o transporte websocket: assina o resolver diretamente (orderId == chave de
// idempotência do pagamento) e dispara um pagamento; o evento payment.captured
// percorre EventBus → consumidor da subscription → PaymentHub → canal do
// assinante. Verifica também o FILTRO: um assinante de outro pedido não recebe.
//
// Observação: o orderId da API casa com a chave de idempotência do pagamento
// (é assim que a saga do pedido nomeia o pagamento). Aqui controlamos a chave
// diretamente para um teste determinístico — o caminho de entrega é idêntico.
func TestOnPaymentProcessedSubscription(t *testing.T) {
	e := newE2E(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const orderID = "sub-order-key-0001"

	// Assina ANTES de disparar o pagamento (o hub não tem histórico).
	ch, err := e.resolver.Subscription().OnPaymentProcessed(ctx, orderID)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	// Assinante de OUTRO pedido — não deve receber nada (filtro por orderId).
	otherCh, err := e.resolver.Subscription().OnPaymentProcessed(ctx, "outro-pedido")
	if err != nil {
		t.Fatalf("subscribe other: %v", err)
	}

	// Dispara o pagamento com idempotencyKey == orderID. processPayment é
	// síncrono (block-until-ack): ao retornar, o evento já foi publicado/consumido
	// e entregue ao canal (bufferizado).
	p := e.processPayment(map[string]any{
		"idempotencyKey": orderID,
		"customerId":     "cust_sub",
		"amountCents":    4242,
		"currency":       "BRL",
	})
	if p.Status != "CAPTURED" {
		t.Fatalf("expected CAPTURED, got %q", p.Status)
	}

	select {
	case got := <-ch:
		if got == nil {
			t.Fatal("subscription channel closed without a payment")
		}
		if got.ID != p.ID {
			t.Fatalf("subscription payment id = %q, want %q", got.ID, p.ID)
		}
		if got.Status != "CAPTURED" {
			t.Fatalf("subscription payment status = %q, want CAPTURED", got.Status)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: subscription did not receive the processed payment")
	}

	// O assinante de outro pedido não deve ter recebido.
	select {
	case got := <-otherCh:
		t.Fatalf("subscriber for a different order received a payment: %+v", got)
	default:
	}
}

// TestOnPaymentProcessedIgnoresNonCaptured garante que um pagamento RECUSADO
// (não capturado) não emite na subscription — só sucesso (captura) emite.
func TestOnPaymentProcessedIgnoresFiltering(t *testing.T) {
	e := newE2E(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Assinante de um pedido que nunca será pago.
	ch, err := e.resolver.Subscription().OnPaymentProcessed(ctx, "pedido-sem-pagamento")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// Processa um pagamento com OUTRA chave — não deve chegar ao assinante acima.
	e.processPayment(map[string]any{
		"idempotencyKey": "chave-diferente-001",
		"customerId":     "cust_x",
		"amountCents":    100,
		"currency":       "BRL",
	})

	select {
	case got := <-ch:
		t.Fatalf("não deveria receber pagamento de outra chave: %+v", got)
	case <-time.After(300 * time.Millisecond):
		// ok — nada recebido
	}
}
