package main

import (
	"context"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/example/payment-federation/internal/app"
)

func main() {
	fx.New(
		// Logger: console colorido em dev, JSON em produção.
		fx.Provide(newLogger),
		// Em dev, os eventos do framework (provided/invoking/...) vão para
		// Debug, ficando fora do nível Info — logs limpos. Em produção saem
		// em Info (JSON), úteis para observabilidade.
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			l := &fxevent.ZapLogger{Logger: log}
			if devMode() {
				l.UseLogLevel(zapcore.DebugLevel)
			}
			return l
		}),
		app.Module,

		// Sobe e desce o servidor HTTP (Fiber) via lifecycle do fx.
		fx.Invoke(registerHTTPServer),
	).Run()
}

// devMode liga o modo de desenvolvimento via APP_ENV (dev/development/local).
// O alvo `make dev` (Air) exporta APP_ENV=dev.
func devMode() bool {
	switch strings.ToLower(os.Getenv("APP_ENV")) {
	case "dev", "development", "local":
		return true
	default:
		return false
	}
}

// newLogger devolve um logger "bonito" (console colorido, timestamp curto) em
// dev, e o logger de produção (JSON estruturado) caso contrário.
func newLogger() (*zap.Logger, error) {
	if !devMode() {
		return zap.NewProduction()
	}
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
	cfg.EncoderConfig.ConsoleSeparator = "  "
	cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	return cfg.Build()
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
