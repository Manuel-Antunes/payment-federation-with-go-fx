package main

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/example/payment-federation/internal/app"
)

func main() {
	fx.New(
		// Logger compartilhado.
		fx.Provide(zap.NewProduction),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),

		// Composição completa: kernel + bounded contexts + interface.
		app.Module,

		// Sobe e desce o servidor HTTP (Fiber) via lifecycle do fx.
		fx.Invoke(registerHTTPServer),
	).Run()
}

func registerHTTPServer(lc fx.Lifecycle, appHTTP *fiber.App, log *zap.Logger) {
	const addr = ":8080"
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				log.Info("HTTP up", zap.String("addr", addr), zap.String("playground", "http://localhost:8080/"))
				if err := appHTTP.Listen(addr); err != nil {
					log.Error("listen", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("HTTP shutting down")
			return appHTTP.ShutdownWithContext(ctx)
		},
	})
}
