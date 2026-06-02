package e2e

import "testing"

func TestOrderRequiresExistingCustomer(t *testing.T) {
	e := newE2E(t)

	if msg := e.mustError(`mutation($in: CreateOrderInput!){ createOrder(input:$in){ id } }`,
		map[string]any{"in": map[string]any{
			"idempotencyKey": "ord-no-cust",
			"customerId":     "usr_does_not_exist",
			"amountCents":    1000,
			"currency":       "BRL",
		}}); msg == "" {
		t.Fatal("expected error creating order for unknown customer")
	}
}

// TestOrderTriggersPaymentSaga valida a saga pedido->pagamento ATRAVÉS da API
// pública: ao criar um pedido, um pagamento é processado automaticamente usando
// o id do pedido como chave de idempotência. Como não há query de pagamento por
// chave, observamos via REPLAY idempotente: um processPayment com a chave ==
// id-do-pedido (e dados propositalmente diferentes) deve devolver o pagamento
// JÁ criado pela saga — com o cliente e o valor do PEDIDO, não os enviados aqui.
func TestOrderTriggersPaymentSaga(t *testing.T) {
	e := newE2E(t)

	customer := e.createUser("Grace", "grace@example.com")

	var co struct {
		CreateOrder struct {
			ID          string `json:"id"`
			CustomerID  string `json:"customerId"`
			AmountCents int64  `json:"amountCents"`
			Status      string `json:"status"`
		} `json:"createOrder"`
	}
	e.mustQuery(`mutation($in: CreateOrderInput!){
  createOrder(input:$in){ id customerId amountCents status }
}`, map[string]any{"in": map[string]any{
		"idempotencyKey": "ord-saga-1",
		"customerId":     customer,
		"amountCents":    7700,
		"currency":       "BRL",
	}}, &co)
	if co.CreateOrder.Status != "CREATED" || co.CreateOrder.AmountCents != 7700 {
		t.Fatalf("unexpected order: %+v", co.CreateOrder)
	}

	// Replay: a chave do pagamento da saga é o ID do pedido.
	replay := e.processPayment(map[string]any{
		"idempotencyKey": co.CreateOrder.ID,
		"customerId":     "someone-else", // ignorado: pagamento já existe
		"amountCents":    1,              // ignorado
		"currency":       "BRL",
	})
	if replay.Status != "CAPTURED" {
		t.Fatalf("saga payment should be CAPTURED, got %q", replay.Status)
	}
	if replay.CustomerID != customer {
		t.Fatalf("saga payment customer mismatch: got %q want %q", replay.CustomerID, customer)
	}
	if replay.AmountCents != 7700 {
		t.Fatalf("saga payment amount mismatch: got %d want 7700 (proves saga, not this call, created it)", replay.AmountCents)
	}
}

// TestOrderCustomerDataLoader exercita o campo Order.customer resolvido via
// DataLoader: cria um usuário, cria um pedido para ele e expande
// order { customer { ... } } — o loader (injetado por requisição) resolve o
// cliente a partir do customerId em lote/cacheado.
func TestOrderCustomerDataLoader(t *testing.T) {
	e := newE2E(t)

	customer := e.createUser("Linus", "linus@example.com")

	var co struct {
		CreateOrder struct{ ID string } `json:"createOrder"`
	}
	e.mustQuery(`mutation($in: CreateOrderInput!){ createOrder(input:$in){ id } }`,
		map[string]any{"in": map[string]any{
			"idempotencyKey": "ord-loader-1",
			"customerId":     customer,
			"amountCents":    3300,
			"currency":       "BRL",
		}}, &co)

	var got struct {
		Order struct {
			ID         string `json:"id"`
			CustomerID string `json:"customerId"`
			Customer   struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"customer"`
		} `json:"order"`
	}
	e.mustQuery(`query($id: ID!){
  order(id:$id){ id customerId customer{ id name email } }
}`, map[string]any{"id": co.CreateOrder.ID}, &got)

	if got.Order.Customer.ID != customer {
		t.Fatalf("dataloader resolved wrong customer: got %q want %q", got.Order.Customer.ID, customer)
	}
	if got.Order.Customer.ID != got.Order.CustomerID {
		t.Fatalf("customer.id (%q) must match order.customerId (%q)", got.Order.Customer.ID, got.Order.CustomerID)
	}
	if got.Order.Customer.Email != "linus@example.com" || got.Order.Customer.Name != "Linus" {
		t.Fatalf("unexpected resolved customer: %+v", got.Order.Customer)
	}
}
