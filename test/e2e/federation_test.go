package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

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
