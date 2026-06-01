package http

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/example/payment-federation/internal/interfaces/graph"
	"github.com/example/payment-federation/internal/interfaces/graph/dataloader"
	"github.com/example/payment-federation/internal/payments/infrastructure/gateway"
)

// NewFiberApp constrói o servidor Fiber e monta:
//   - POST /query           : endpoint GraphQL (subgraph federado)
//   - GET  /                : GraphQL Playground
//   - POST /admin/gateway   : troca o gateway ativo EM RUNTIME
//   - GET  /admin/gateway   : lista gateways e o ativo
//
// O handler GraphQL é envolvido pelo middleware de DataLoaders, que injeta um
// conjunto fresco de loaders por requisição (batch + cache por-request).
func NewFiberApp(resolver *graph.Resolver, loaders dataloader.Middleware, registry *gateway.Registry) *fiber.App {
	app := fiber.New(fiber.Config{AppName: "payment-federation"})
	app.Use(recover.New())

	// gqlgen é um http.Handler; adaptamos para o Fiber. O schema executável é
	// montado no pacote graph (com os resolvers + a diretiva @binding).
	es := graph.NewExecutableSchema(resolver)
	gql := handler.NewDefaultServer(es)
	app.All("/query", adaptor.HTTPHandler(loaders(gql)))
	app.Get("/", adaptor.HTTPHandler(playground.Handler("Payment Federation", "/query")))

	registerAdmin(app, registry)
	return app
}

func registerAdmin(app *fiber.App, registry *gateway.Registry) {
	admin := app.Group("/admin/gateway")

	admin.Get("/", func(c *fiber.Ctx) error {
		var active string
		if g := registry.Active(); g != nil {
			active = g.Name()
		}
		return c.JSON(fiber.Map{"active": active, "available": registry.Available()})
	})

	// Troca o provedor padrão sem reiniciar o serviço.
	admin.Post("/", func(c *fiber.Ctx) error {
		var body struct {
			Name string `json:"name"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if err := registry.SetActive(body.Name); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(fiber.Map{"active": body.Name})
	})
}
