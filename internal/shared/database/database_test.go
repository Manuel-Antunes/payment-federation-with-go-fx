package database

import "testing"

func TestDSNDefaultsToLucid(t *testing.T) {
	for _, k := range []string{"DATABASE_URL", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE"} {
		t.Setenv(k, "")
	}
	got := DSN()
	want := "postgres://lucid:lucid@localhost:5432/lucid?sslmode=disable"
	if got != want {
		t.Fatalf("DSN() default = %q, want %q", got, want)
	}
}

func TestDSNFromIndividualVars(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_USER", "alice")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("DB_NAME", "shop")
	t.Setenv("DB_SSLMODE", "require")
	got := DSN()
	want := "postgres://alice:secret@db:6543/shop?sslmode=require"
	if got != want {
		t.Fatalf("DSN() = %q, want %q", got, want)
	}
}

func TestDSNDatabaseURLOverrides(t *testing.T) {
	t.Setenv("DB_USER", "ignored")
	t.Setenv("DATABASE_URL", "postgres://override/here")
	if got := DSN(); got != "postgres://override/here" {
		t.Fatalf("DATABASE_URL should win, got %q", got)
	}
}
