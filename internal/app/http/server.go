// Package http monta o handler HTTP da aplicação com a biblioteca padrão
// (net/http). Foi escolhido net/http em vez do adaptador do Fiber porque o
// transporte WebSocket do gqlgen (subscriptions) precisa de http.Hijacker — que
// o adaptador fasthttp não expõe.
package http

import (
	"encoding/json"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/example/payment-federation/internal/app/graph"
	"github.com/example/payment-federation/internal/app/graph/dataloader"
	"github.com/example/payment-federation/internal/payments/infrastructure/gateway"
)

// NewHandler monta o http.Handler do servidor e mapeia:
//   - POST/GET/WS /query   : GraphQL (queries, mutations e SUBSCRIPTIONS via WS)
//   - GET  /               : GraphQL Playground (com suporte a subscriptions)
//   - GET  /admin/gateway  : lista gateways e o ativo
//   - POST /admin/gateway  : troca o gateway ativo EM RUNTIME
//
// O handler GraphQL é envolvido pelo middleware de DataLoaders, que injeta um
// conjunto fresco de loaders por requisição (batch + cache por-request).
func NewHandler(resolver *graph.Resolver, loaders dataloader.Middleware, registry *gateway.Registry) http.Handler {
	// NewDefaultServer já inclui o transporte WebSocket (subscriptions).
	es := graph.NewExecutableSchema(resolver)
	gql := handler.NewDefaultServer(es)

	mux := http.NewServeMux()
	// O Playground aponta para /query; o GraphQL Playground usa o mesmo endpoint
	// em ws:// para as subscriptions.
	mux.Handle("/query", loaders(gql))
	mux.Handle("/", playground.Handler("Payment Federation", "/query"))
	registerAdmin(mux, registry)
	return mux
}

func registerAdmin(mux *http.ServeMux, registry *gateway.Registry) {
	mux.HandleFunc("/admin/gateway", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			var active string
			if g := registry.Active(); g != nil {
				active = g.Name()
			}
			writeJSON(w, http.StatusOK, map[string]any{"active": active, "available": registry.Available()})

		case http.MethodPost:
			// Troca o provedor padrão sem reiniciar o serviço.
			var body struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			if err := registry.SetActive(body.Name); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"active": body.Name})

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
