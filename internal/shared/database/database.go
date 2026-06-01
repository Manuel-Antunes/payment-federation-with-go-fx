// Package database fornece a conexão Postgres COMPARTILHADA (*sql.DB) usada por
// todos os módulos. Cada módulo cria o seu próprio *ent.Client por cima desta
// mesma conexão (entsql.OpenDB), então há um único pool, mas schemas/clients
// independentes por bounded context.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq" // driver "postgres"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DSN resolve a string de conexão. Prioriza DATABASE_URL (sobrescreve tudo);
// caso contrário, monta a partir das variáveis individuais, com defaults
// "lucid" para usuário/senha/banco (dev local / docker-compose). As variáveis
// vêm do ambiente — carregue um .env com godotenv no boot (ver cmd/server).
func DSN() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		getenv("DB_USER", "lucid"),
		getenv("DB_PASSWORD", "lucid"),
		getenv("DB_HOST", "localhost"),
		getenv("DB_PORT", "5432"),
		getenv("DB_NAME", "lucid"),
		getenv("DB_SSLMODE", "disable"),
	)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// NewDB abre o pool Postgres e pendura o ciclo de vida no fx (ping no start,
// close no stop). O DSN vem do ambiente.
func NewDB(lc fx.Lifecycle, log *zap.Logger) (*sql.DB, error) {
	db, err := sql.Open("postgres", DSN())
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := db.PingContext(ctx); err != nil {
				return err
			}
			log.Info("postgres conectado")
			return nil
		},
		OnStop: func(context.Context) error {
			return db.Close()
		},
	})
	return db, nil
}
