// Package http monta o servidor HTTP da aplicação com o Fiber (fasthttp).
//
// As queries/mutations GraphQL e o Playground são servidos pelo handler do
// gqlgen, adaptado para o fasthttp (adaptor.HTTPHandler). As SUBSCRIPTIONS usam
// um handler graphql-transport-ws PRÓPRIO sobre o WebSocket do Fiber
// (gofiber/contrib/websocket), dirigindo o executor do gqlgen — porque o
// transporte WS embutido do gqlgen depende de http.Hijacker (net/http), que o
// fasthttp não expõe. Ver graphql_ws.go.
package http

import (
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"go.uber.org/zap"

	"github.com/example/payment-federation/internal/app/graph"
	"github.com/example/payment-federation/internal/app/graph/dataloader"
	"github.com/example/payment-federation/internal/payments/infrastructure/gateway"
	"github.com/example/payment-federation/internal/shared/cqrs"
)

// NewFiberApp monta o *fiber.App e mapeia:
//   - POST /query          : GraphQL (queries, mutations)
//   - GET  /query (upgrade): SUBSCRIPTIONS via WebSocket (graphql-transport-ws)
//   - GET  /               : GraphQL Playground (com suporte a subscriptions)
//   - GET/POST /admin/gateway : lista / troca o gateway ativo em runtime
//
// O handler HTTP do GraphQL é envolvido pelo middleware de DataLoaders (um
// conjunto fresco por requisição) e adaptado para o fasthttp.
func NewFiberApp(
	resolver *graph.Resolver,
	loaders dataloader.Middleware,
	registry *gateway.Registry,
	queries *cqrs.QueryBus,
	log *zap.Logger,
) *fiber.App {
	es := graph.NewExecutableSchema(resolver)
	gql := handler.NewDefaultServer(es) // queries/mutations (o transporte WS dele não é usado)
	exec := executor.New(es)            // executor dirigido pelo handler WS próprio

	// Handler HTTP do GraphQL (com loaders) adaptado para o fasthttp.
	httpGQL := adaptor.HTTPHandler(loaders(gql))

	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	// /query: se for upgrade WebSocket -> handler de subscriptions; senão -> HTTP.
	app.Use("/query", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return httpGQL(c)
	})
	app.Get("/query", websocket.New(
		newGraphQLWSHandler(exec, queries, log),
		websocket.Config{Subprotocols: []string{"graphql-transport-ws"}},
	))

	app.Get("/", adaptor.HTTPHandler(playground.Handler("Payment Federation", "/query")))
	registerAdmin(app, registry)
	return app
}

func registerAdmin(app *fiber.App, registry *gateway.Registry) {
	app.Get("/admin/gateway", func(c *fiber.Ctx) error {
		var active string
		if g := registry.Active(); g != nil {
			active = g.Name()
		}
		return c.JSON(fiber.Map{"active": active, "available": registry.Available()})
	})

	app.Post("/admin/gateway", func(c *fiber.Ctx) error {
		// Troca o provedor padrão sem reiniciar o serviço.
		var body struct {
			Name string `json:"name"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if err := registry.SetActive(body.Name); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"active": body.Name})
	})
}
