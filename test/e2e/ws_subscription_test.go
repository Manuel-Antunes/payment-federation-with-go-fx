package e2e

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestOnPaymentProcessedOverWebsocket prova o transporte de SUBSCRIPTION pela
// REDE (WebSocket), falando o protocolo graphql-transport-ws — o mesmo que o
// Playground usa. Agora roda sobre o Fiber (fasthttp): o gqlgen não consegue
// servir WS no fasthttp (precisa de http.Hijacker), então o servidor usa um
// handler graphql-transport-ws próprio sobre o gofiber/contrib/websocket,
// dirigindo o executor do gqlgen. O cliente abaixo é o gorilla (lado cliente é
// agnóstico ao servidor).
func TestOnPaymentProcessedOverWebsocket(t *testing.T) {
	e := newE2E(t)
	baseURL := e.serveWS() // WS precisa de listener real (app.Test é em memória)

	d := websocket.Dialer{Subprotocols: []string{"graphql-transport-ws"}}
	conn, _, err := d.Dial("ws"+strings.TrimPrefix(baseURL, "http")+"/query", nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	// Handshake graphql-transport-ws: init -> ack.
	wsSend(t, conn, map[string]any{"type": "connection_init"})
	if got := wsReadType(t, conn); got != "connection_ack" {
		t.Fatalf("esperava connection_ack, veio %q", got)
	}

	const orderID = "ws-sub-order-0001"
	wsSend(t, conn, map[string]any{
		"id":   "1",
		"type": "subscribe",
		"payload": map[string]any{
			"query": fmt.Sprintf(`subscription { onPaymentProcessed(orderId: %q){ id status amountCents } }`, orderID),
		},
	})

	// Espera o servidor registrar a subscription antes de disparar (o hub não
	// guarda histórico) — determinístico, sem sleep arbitrário.
	waitUntil(t, 2*time.Second, func() bool { return e.resolver.Payments.NumSubscribers() >= 1 })

	p := e.processPayment(map[string]any{
		"idempotencyKey": orderID,
		"customerId":     "cust_ws",
		"amountCents":    4242,
		"currency":       "BRL",
	})
	if p.Status != "CAPTURED" {
		t.Fatalf("expected CAPTURED, got %q", p.Status)
	}

	// Lê o próximo "next" da subscription (ignora ping/keep-alive).
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var msg struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read websocket: %v", err)
		}
		switch msg.Type {
		case "next":
			var pl struct {
				Data struct {
					OnPaymentProcessed struct {
						ID          string `json:"id"`
						Status      string `json:"status"`
						AmountCents int64  `json:"amountCents"`
					} `json:"onPaymentProcessed"`
				} `json:"data"`
			}
			if err := json.Unmarshal(msg.Payload, &pl); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			got := pl.Data.OnPaymentProcessed
			if got.ID != p.ID || got.Status != "CAPTURED" || got.AmountCents != 4242 {
				t.Fatalf("unexpected payload over WS: %+v", got)
			}
			return
		case "error":
			t.Fatalf("subscription error over WS: %s", msg.Payload)
		default: // ping / pong / connection_ack tardio: ignora
			continue
		}
	}
}

func wsSend(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	if err := conn.WriteJSON(v); err != nil {
		t.Fatalf("write websocket: %v", err)
	}
}

func wsReadType(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var msg struct {
		Type string `json:"type"`
	}
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read websocket: %v", err)
	}
	return msg.Type
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timeout esperando condição")
}
