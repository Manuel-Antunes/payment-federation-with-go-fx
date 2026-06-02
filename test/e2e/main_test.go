package e2e

// Bootstrap da suíte e2e: UM container Postgres (testcontainers) por suíte, com
// uma conexão admin aberta UMA vez. Os bancos de cada teste são criados/dropados
// pelo harness (ver harness_test.go).

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	pgUser = "test"
	pgPass = "test"
)

// Estado da suíte: o container e a conexão admin (ao banco "postgres") são
// criados UMA vez em TestMain; cada teste cria/dropa o seu próprio banco.
var (
	pgHost    string
	pgPort    string
	adminDB   *sql.DB
	dbCounter atomic.Int64
)

func dsn(dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", pgUser, pgPass, pgHost, pgPort, dbName)
}

// TestMain sobe um único Postgres para toda a suíte e abre a conexão admin
// usada para criar/dropar os bancos de cada teste.
func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("postgres"),
		postgres.WithUsername(pgUser),
		postgres.WithPassword(pgPass),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testcontainers: subir postgres: %v\n", err)
		os.Exit(1)
	}

	host, err := container.Host(ctx)
	mustNoErr("host", err)
	port, err := container.MappedPort(ctx, "5432/tcp")
	mustNoErr("port", err)
	pgHost, pgPort = host, port.Port()

	adminDB, err = sql.Open("postgres", dsn("postgres"))
	mustNoErr("open admin", err)
	// O wait do módulo já garante prontidão, mas pingamos com retry por segurança.
	for i := 0; i < 30; i++ {
		if adminDB.PingContext(ctx) == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	code := m.Run()

	_ = adminDB.Close()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

func mustNoErr(what string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "testcontainers: %s: %v\n", what, err)
		os.Exit(1)
	}
}

// dropDatabase remove o banco do teste (FORCE encerra conexões remanescentes).
func dropDatabase(name string) {
	_, _ = adminDB.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)")
}
