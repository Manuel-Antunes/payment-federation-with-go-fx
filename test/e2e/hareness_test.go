package e2e

// Utilitários de teste e2e: sobe uma instância isolada do app (Fiber + gqlgen +
// fx) por teste, num listener TCP real (preciso para o WebSocket das
// subscriptions), ligada a um banco Postgres próprio (criado/migrado/dropado
// pelo harness), e expõe helpers de transporte GraphQL/HTTP + fixtures.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/example/payment-federation/internal/app"
	"github.com/example/payment-federation/internal/app/graph"
)

type e2e struct {
	t        *testing.T
	app      *fiber.App      // testado em memória via app.Test (HTTP)
	resolver *graph.Resolver // exposto p/ testar subscriptions sem websocket
}

// newE2E cria um BANCO novo no container compartilhado, sobe uma instância
// isolada do app conectada a ele (a migração roda no Start do fx) e registra o
// teardown (stop do app + close + DROP DATABASE) via t.Cleanup. As requisições
// HTTP rodam em memória (app.Test) — sem bind de porta; o WebSocket usa um
// listener real, criado sob demanda por serveWS.
func newE2E(t *testing.T) *e2e {
	t.Helper()

	dbName := fmt.Sprintf("e2e_%d", dbCounter.Add(1))
	if _, err := adminDB.Exec("CREATE DATABASE " + dbName); err != nil {
		t.Fatalf("create database %s: %v", dbName, err)
	}

	db, err := sql.Open("postgres", dsn(dbName))
	if err != nil {
		t.Fatalf("open db %s: %v", dbName, err)
	}

	var fiberApp *fiber.App
	var resolver *graph.Resolver
	fxApp := fx.New(
		fx.Provide(func() *zap.Logger { return zap.NewNop() }),
		fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
		fx.Supply(db), // injeta o *sql.DB do teste; os clients ent migram no Start
		app.Module,
		fx.Populate(&fiberApp, &resolver),
	)

	startCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := fxApp.Start(startCtx); err != nil {
		_ = db.Close()
		dropDatabase(dbName)
		t.Fatalf("start app: %v", err)
	}

	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopCancel()
		_ = fxApp.Stop(stopCtx)
		_ = db.Close()
		dropDatabase(dbName)
	})

	return &e2e{t: t, app: fiberApp, resolver: resolver}
}

// serveWS sobe um listener TCP real e serve o Fiber nele, devolvendo o baseURL.
// Necessário só para o WebSocket (subscriptions): o app.Test é em memória e não
// expõe uma conexão dialável para o upgrade WS. O shutdown é registrado no
// t.Cleanup.
func (e *e2e) serveWS() string {
	e.t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		e.t.Fatalf("listen: %v", err)
	}
	go func() { _ = e.app.Listener(ln) }()

	base := "http://" + ln.Addr().String()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, derr := http.Get(base + "/")
		if derr == nil {
			_ = resp.Body.Close()
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Fecha só o listener (encerra o Serve). NÃO usamos app.ShutdownWithContext:
	// ele escreve no estado do fasthttp.Server concorrentemente com RequestCtx
	// ainda vivos das requisições em memória (app.Test) -> data race no shutdown.
	e.t.Cleanup(func() { _ = ln.Close() })
	return base
}

// --- transporte GraphQL / HTTP ----------------------------------------------

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

	req, err := http.NewRequest(http.MethodPost, "/query", bytes.NewReader(body))
	if err != nil {
		e.t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.app.Test(req, -1) // -1: sem timeout (espera o handler completar)
	if err != nil {
		e.t.Fatalf("app.Test query: %v", err)
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
	req, err := http.NewRequest(method, path, r)
	if err != nil {
		e.t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := e.app.Test(req, -1)
	if err != nil {
		e.t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
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
