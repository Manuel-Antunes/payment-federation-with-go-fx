.PHONY: tidy generate test e2e test-all run dev db-up db-down

AIR_VERSION ?= v1.65.3

# Baixa dependências e gera go.sum (requer código gerado por ent + gqlgen)
tidy: generate
	go mod tidy

# Gera o código: ent (por módulo), gqlgen e, por fim, os DataLoaders.
# Ordem importa: o dataloaden carrega o pacote `model` (gerado pelo gqlgen),
# então roda DEPOIS do gqlgen.
generate:
	GOFLAGS=-mod=mod go mod download
	GOFLAGS=-mod=mod go generate ./internal/user/infrastructure/ent/ ./internal/payments/infrastructure/ent/
	GOFLAGS=-mod=mod go run github.com/99designs/gqlgen generate
	GOFLAGS=-mod=mod go generate ./internal/app/graph/dataloader/

# Sobe/derruba o Postgres local (docker compose).
db-up:
	docker compose up -d postgres

db-down:
	docker compose down

# Testes de domínio (não dependem de código gerado nem de banco)
test:
	go test ./internal/payments/domain/... ./internal/user/domain/...

# Testes end-to-end (sobem o app via fx + Postgres via testcontainers; requerem
# Docker e `make generate`). Vivem em ./test/e2e.
e2e: generate
	GOFLAGS=-mod=mod go test -race ./test/e2e/...

# Tudo: domínio + e2e (requer código gerado + Docker).
test-all: generate
	GOFLAGS=-mod=mod go test -race ./internal/... ./test/...

# Sobe o servidor (requer `make generate` antes, na primeira vez)
run:
	go run ./cmd/server

# Desenvolvimento local com live reload (Air) e logs bonitos (APP_ENV=dev liga
# o logger de console colorido). Gera o gqlgen primeiro e usa o `air` global se
# existir; senão, roda via `go run` (sem instalar nada).
dev: generate
	@if command -v air >/dev/null 2>&1; then \
		APP_ENV=dev air; \
	else \
		echo ">> air não encontrado no PATH; usando 'go run github.com/air-verse/air@$(AIR_VERSION)'"; \
		APP_ENV=dev go run github.com/air-verse/air@$(AIR_VERSION); \
	fi

build:
	go build -o payment-federation ./cmd/server