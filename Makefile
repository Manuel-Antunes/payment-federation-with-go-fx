.PHONY: tidy generate test run dev

AIR_VERSION ?= v1.65.3

# Baixa dependências e gera go.sum (requer código gerado pelo gqlgen)
tidy: generate
	go mod tidy

# Gera o código do gqlgen (generated.go, federation.go, models_gen.go)
generate:
	GOFLAGS=-mod=mod go mod download
	GOFLAGS=-mod=mod go run github.com/99designs/gqlgen generate

# Testes de domínio (não dependem do código gerado pelo gqlgen)
test:
	go test ./internal/payments/domain/...

# Sobe o servidor (requer `make generate` antes, na primeira vez)
run:
	go run ./cmd/server

# Desenvolvimento local com live reload (Air). Gera o código do gqlgen primeiro
# e usa o `air` global se existir; senão, roda via `go run` (sem instalar nada).
dev: generate
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo ">> air não encontrado no PATH; usando 'go run github.com/air-verse/air@$(AIR_VERSION)'"; \
		go run github.com/air-verse/air@$(AIR_VERSION); \
	fi
