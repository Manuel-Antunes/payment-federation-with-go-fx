package http_test

// Testes END-TO-END: exercitam o servidor real (Fiber + gqlgen + fx) do ponto
// de entrada HTTP (POST /query, /admin/gateway) até a resposta, passando por
// toda a stack — resolvers, barramentos de CQRS (Watermill), use-cases, domínio
// e repositórios in-memory. Cada teste sobe uma instância isolada da aplicação
// (sem escutar porta: usa fiber.App.Test em memória), então estado não vaza
// entre testes.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/example/payment-federation/internal/app"
)

// testTimeoutMS limita cada request (em ms) para que um deadlock falhe o teste
// em vez de travar. O caminho da saga (block-until-ack) roda em memória, bem
// abaixo disso.
const testTimeoutMS = 10_000

// --- harness ---------------------------------------------------------------

type e2e struct {
	t   *testing.T
	app *fiber.App
}

// newE2E sobe uma instância isolada do app e registra o teardown via t.Cleanup.
func newE2E(t *testing.T) *e2e {
	t.Helper()

	var fiberApp *fiber.App
	fxApp := fx.New(
		fx.Provide(func() *zap.Logger { return zap.NewNop() }),
		fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
		app.Module,
		fx.Populate(&fiberApp),
	)

	startCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := fxApp.Start(startCtx); err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		_ = fxApp.Stop(stopCtx)
	})

	return &e2e{t: t, app: fiberApp}
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
		Path    []any  `json:"path"`
	} `json:"errors"`
}

func (r gqlResponse) errorMessages() string {
	msgs := make([]string, 0, len(r.Errors))
	for _, e := range r.Errors {
		msgs = append(msgs, e.Message)
	}
	return strings.Join(msgs, "; ")
}

// query envia uma operação GraphQL e devolve a resposta crua (data + errors),
// SEM falhar em erro de GraphQL — para que testes de caminho-feliz e de erro
// usem o mesmo transporte.
func (e *e2e) query(operation string, variables map[string]any) gqlResponse {
	e.t.Helper()

	payload := map[string]any{"query": operation}
	if variables != nil {
		payload["variables"] = variables
	}
	body, err := json.Marshal(payload)
	if err != nil {
		e.t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.app.Test(req, testTimeoutMS)
	if err != nil {
		e.t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("read response: %v", err)
	}
	// O endpoint sempre responde 200 em GraphQL (erros vão no corpo).
	if resp.StatusCode != http.StatusOK {
		e.t.Fatalf("unexpected status %d: %s", resp.StatusCode, raw)
	}

	var out gqlResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		e.t.Fatalf("decode response: %v (body: %s)", err, raw)
	}
	return out
}

// mustQuery executa uma operação que deve ter SUCESSO e decodifica `data`.
func (e *e2e) mustQuery(operation string, variables map[string]any, out any) {
	e.t.Helper()
	resp := e.query(operation, variables)
	if len(resp.Errors) > 0 {
		e.t.Fatalf("unexpected graphql error: %s", resp.errorMessages())
	}
	if out != nil {
		if err := json.Unmarshal(resp.Data, out); err != nil {
			e.t.Fatalf("decode data: %v (data: %s)", err, resp.Data)
		}
	}
}

// mustError executa uma operação que deve FALHAR e devolve a mensagem de erro.
func (e *e2e) mustError(operation string, variables map[string]any) string {
	e.t.Helper()
	resp := e.query(operation, variables)
	if len(resp.Errors) == 0 {
		e.t.Fatalf("expected graphql error, got data: %s", resp.Data)
	}
	return resp.errorMessages()
}

// httpJSON faz uma chamada HTTP não-GraphQL (endpoints admin) e devolve status e corpo.
func (e *e2e) httpJSON(method, path, body string) (int, string) {
	e.t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := e.app.Test(req, testTimeoutMS)
	if err != nil {
		e.t.Fatalf("request %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

// --- fixtures / fragments ---------------------------------------------------

const userFields = `id name email`

func (e *e2e) createUser(name, email string) string {
	e.t.Helper()
	var out struct {
		CreateUser struct{ ID, Name, Email string } `json:"createUser"`
	}
	e.mustQuery(`mutation($in: CreateUserInput!){ createUser(input:$in){ `+userFields+` } }`,
		map[string]any{"in": map[string]any{"name": name, "email": email}}, &out)
	if out.CreateUser.ID == "" {
		e.t.Fatal("createUser returned empty id")
	}
	return out.CreateUser.ID
}

type paymentView struct {
	ID            string `json:"id"`
	CustomerID    string `json:"customerId"`
	AmountCents   int64  `json:"amountCents"`
	Currency      string `json:"currency"`
	RefundedCents int64  `json:"refundedCents"`
	Status        string `json:"status"`
}

func (e *e2e) processPayment(in map[string]any) paymentView {
	e.t.Helper()
	var out struct {
		ProcessPayment paymentView `json:"processPayment"`
	}
	e.mustQuery(`mutation($in: ProcessPaymentInput!){
  processPayment(input:$in){ id customerId amountCents currency refundedCents status }
}`, map[string]any{"in": in}, &out)
	return out.ProcessPayment
}

// --- user --------------------------------------------------------------------

func TestUserMutationsAndQueries(t *testing.T) {
	e := newE2E(t)

	id := e.createUser("Ada Lovelace", "ada@example.com")

	// query por id
	var byID struct {
		User struct{ ID, Name, Email string } `json:"user"`
	}
	e.mustQuery(`query($id: ID!){ user(id:$id){ `+userFields+` } }`, map[string]any{"id": id}, &byID)
	if byID.User.Email != "ada@example.com" || byID.User.Name != "Ada Lovelace" {
		t.Fatalf("unexpected user: %+v", byID.User)
	}

	// listagem
	var list struct {
		Users []struct{ ID string } `json:"users"`
	}
	e.mustQuery(`query{ users{ id } }`, nil, &list)
	if len(list.Users) != 1 || list.Users[0].ID != id {
		t.Fatalf("expected exactly the created user in list, got %+v", list.Users)
	}

	// update (nome + email) e leitura refletindo a mudança
	var upd struct {
		UpdateUser struct{ ID, Name, Email string } `json:"updateUser"`
	}
	e.mustQuery(`mutation($in: UpdateUserInput!){ updateUser(input:$in){ `+userFields+` } }`,
		map[string]any{"in": map[string]any{"id": id, "name": "Ada B.", "email": "ada.b@example.com"}}, &upd)
	if upd.UpdateUser.Name != "Ada B." || upd.UpdateUser.Email != "ada.b@example.com" {
		t.Fatalf("update not reflected: %+v", upd.UpdateUser)
	}
}

func TestUserValidationAndIdempotency(t *testing.T) {
	e := newE2E(t)

	// e-mail inválido => erro de domínio até a borda.
	if msg := e.mustError(`mutation($in: CreateUserInput!){ createUser(input:$in){ id } }`,
		map[string]any{"in": map[string]any{"name": "X", "email": "not-an-email"}}); msg == "" {
		t.Fatal("expected error message for invalid email")
	}

	// nome vazio => erro.
	if msg := e.mustError(`mutation($in: CreateUserInput!){ createUser(input:$in){ id } }`,
		map[string]any{"in": map[string]any{"name": "  ", "email": "ok@example.com"}}); msg == "" {
		t.Fatal("expected error message for empty name")
	}

	// updateUser de id inexistente => erro.
	if msg := e.mustError(`mutation($in: UpdateUserInput!){ updateUser(input:$in){ id } }`,
		map[string]any{"in": map[string]any{"id": "usr_missing", "name": "Z"}}); msg == "" {
		t.Fatal("expected error for unknown user update")
	}

	// e-mail único: criar com o mesmo e-mail devolve o usuário existente
	// (read-after-write pela chave natural), sem duplicar.
	id1 := e.createUser("First", "dup@example.com")
	id2 := e.createUser("Second", "dup@example.com")
	if id1 != id2 {
		t.Fatalf("expected same user id for duplicate email, got %s and %s", id1, id2)
	}
	var list struct {
		Users []struct{ ID string } `json:"users"`
	}
	e.mustQuery(`query{ users{ id } }`, nil, &list)
	if len(list.Users) != 1 {
		t.Fatalf("duplicate email must not create a second user, got %d", len(list.Users))
	}
}

// --- payment -----------------------------------------------------------------

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

// --- order + saga ------------------------------------------------------------

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

// --- federation + admin ------------------------------------------------------

func TestFederationServiceSDL(t *testing.T) {
	e := newE2E(t)

	var out struct {
		Service struct {
			SDL string `json:"sdl"`
		} `json:"_service"`
	}
	e.mustQuery(`query{ _service{ sdl } }`, nil, &out)
	for _, want := range []string{"type Payment", "type Order", "type User", "@key"} {
		if !strings.Contains(out.Service.SDL, want) {
			t.Fatalf("federation SDL missing %q", want)
		}
	}
}

func TestAdminGatewaySwitch(t *testing.T) {
	e := newE2E(t)

	// estado inicial: mock ativo, mock+stripe disponíveis.
	status, body := e.httpJSON(http.MethodGet, "/admin/gateway", "")
	if status != http.StatusOK {
		t.Fatalf("GET /admin/gateway status %d: %s", status, body)
	}
	var initial struct {
		Active    string   `json:"active"`
		Available []string `json:"available"`
	}
	if err := json.Unmarshal([]byte(body), &initial); err != nil {
		t.Fatalf("decode admin response: %v", err)
	}
	if initial.Active != "mock" {
		t.Fatalf("expected active gateway 'mock', got %q", initial.Active)
	}
	if !contains(initial.Available, "mock") || !contains(initial.Available, "stripe") {
		t.Fatalf("expected mock and stripe available, got %v", initial.Available)
	}

	// troca em runtime para stripe.
	if status, body := e.httpJSON(http.MethodPost, "/admin/gateway", `{"name":"stripe"}`); status != http.StatusOK {
		t.Fatalf("POST switch status %d: %s", status, body)
	}
	_, body = e.httpJSON(http.MethodGet, "/admin/gateway", "")
	var after struct {
		Active string `json:"active"`
	}
	_ = json.Unmarshal([]byte(body), &after)
	if after.Active != "stripe" {
		t.Fatalf("expected active gateway 'stripe' after switch, got %q", after.Active)
	}

	// trocar para um gateway inexistente é rejeitado.
	if status, _ := e.httpJSON(http.MethodPost, "/admin/gateway", `{"name":"nope"}`); status == http.StatusOK {
		t.Fatal("expected error switching to unknown gateway")
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
