# payment-federation

Exemplo de serviço de **processamento de pagamentos idempotente** em Go, modelado com **DDD + CQRS**, injeção de dependências com **Uber FX**, HTTP com **Fiber** e GraphQL **federado** (Apollo Federation v2) gerado por **gqlgen**.

Dois princípios guiam a arquitetura:

1. **Toda regra de negócio vive no domínio e é aplicada ANTES de qualquer persistência.** O agregado `Payment` valida invariantes e transições de estado em memória; o repositório só grava um agregado já considerado válido.
2. **O gateway de pagamento é um serviço genérico, substituível em runtime.** A aplicação depende da porta `payment.Gateway` / `payment.GatewayProvider`, nunca de um provedor concreto. Um `Registry` permite trocar o provedor ativo sem reiniciar o serviço.

## Arquitetura (monólito modular)

O `internal/` é dividido em **módulos**, não em camadas globais: `shared/` (kernel
reutilizável), `user/` e `payments/` (bounded contexts), e `interfaces/` + `app/`
(composição). Cada bounded context tem suas próprias camadas DDD e é dividido em
**três `module.go` de fx**: `application/module.go` (registro de commands/queries/
events), `infrastructure/module.go` (repos, gateways, bindings de IoC das portas) e o
`module.go` raiz que une os dois. Adicionar um contexto novo não toca nos existentes.

```
cmd/server                          # bootstrap (fx): app.Module + lifecycle HTTP
internal/
├── shared/                         # KERNEL compartilhado (não conhece os contextos)
│   ├── clock/                      #   Clock + SystemClock
│   ├── cqrs/                       #   mediator GENÉRICO (CommandBus/QueryBus, generics)
│   ├── messaging/                  #   backbone Watermill: Command/Event Bus+Processor, router
│   └── module.go                   #   shared.Module
├── user/                           # BOUNDED CONTEXT de usuário
│   ├── domain/user/                #   Aggregate User, VOs (Email), portas
│   ├── application/                #   ports/, command (Create/Update), query (Get/ByEmail/List/Exists)
│   │   ├── bus/                    #     registro de commands+queries nos barramentos
│   │   ├── module.go               #     application.Module (registra commands/queries)
│   │   └── port/                   #     portas (IDGenerator) — subpacote evita ciclo
│   ├── infrastructure/             #   persistence + adapters
│   │   └── module.go               #     infrastructure.Module (repos + bindings)
│   └── module.go                   #   user.Module = application.Module + infrastructure.Module
├── payments/                       # BOUNDED CONTEXT de pagamentos (inclui Order)
│   ├── domain/payment/             #   Aggregate Payment, VOs (Money/Currency/IdempotencyKey)
│   ├── domain/order/               #   Aggregate Order + evento OrderCreated
│   ├── application/
│   │   ├── port/                   #     IDGenerator, EventPublisher, OrderEventPublisher,
│   │   │                           #     OrderIDGenerator, CustomerDirectory (porta cross-module)
│   │   ├── command/                #     ProcessPayment, RefundPayment, CreateOrder
│   │   ├── query/                  #     GetPayment(ByKey), GetOrder(ByKey)
│   │   ├── event/                  #     eventos de integração (Payment/Order) + mapeamento
│   │   ├── bus/                    #     registro de commands/queries/events (inclui a SAGA)
│   │   └── module.go               #     application.Module
│   ├── infrastructure/
│   │   ├── persistence/ gateway/ adapters/ messaging/   # repos, gateways, publishers Watermill
│   │   └── module.go               #     infrastructure.Module
│   └── module.go                   #   payments.Module
├── interfaces/                     # INTERFACE única (um subgraph federado)
│   ├── graph/                      #   schema (payment+order+user), resolvers, model (gqlgen)
│   └── http/                       #   servidor Fiber (/query, playground, /admin/gateway)
└── app/                            # COMPOSIÇÃO: app.Module + bridges cross-module
    ├── app.go                      #   une shared+user+payments+interfaces
    └── bridges.go                  #   customerDirectory (payments↔user via QueryBus)
```

A direção das dependências aponta **para dentro**: `interfaces`/`infrastructure`
dependem de `application`/`domain`, nunca o contrário; os contextos dependem de
`shared/`, nunca o contrário; e um contexto **nunca importa o outro** — a integração
cross-module (pedido exige um cliente existente) passa por uma porta (`CustomerDirectory`)
cujo bridge vive na composição (`app/`).

### CQRS com Watermill

- **CommandBus genérico** (`shared/cqrs`): registry por tipo (Go generics) que a interface
  usa. Cada comando é encaminhado ao **CommandBus do Watermill** (`CommandProcessor` roteia
  ao use-case, que persiste e publica eventos).
- **EventBus** (Watermill): `WatermillPublisher`/`WatermillOrderPublisher` publicam os
  eventos de domínio; `EventProcessor` os consome (projeções e **sagas**).
- **QueryBus genérico** (`shared/cqrs`): síncrono e in-process.

O transporte padrão é o **GoChannel** in-memory com `BlockPublishUntilSubscriberAck`:
`Send`/`Publish` bloqueiam até o Ack, então a escrita assíncrona se comporta de forma
síncrona o bastante para o **read-after-write** das mutations. Trocar por Kafka/NATS/SQS
é só substituir o Pub/Sub.

### Saga: pedido → pagamento

`createOrder` exige um `customerId` de um **usuário existente** (validado via
`CustomerDirectory`). Ao persistir, o pedido emite **OrderCreated** no EventBus; um
`EventHandler` (saga) consome o evento e **dispara automaticamente o ProcessPayment**,
usando o id do pedido como chave de idempotência:

```
createOrder ──cmd──▶ CreateOrder handler ──persist──▶ publica OrderCreated
                                                          │ (EventBus)
                                                          ▼
                          saga: OrderCreated ──cmd──▶ ProcessPayment ──▶ eventos de pagamento
```

## Fluxo de um pagamento (idempotente)

```
processPayment(input)
  └─ valida VOs (Money, Currency, IdempotencyKey)         [domínio]
  └─ existe pagamento p/ a IdempotencyKey? → devolve (replay)
  └─ NewPayment(...)                                       [domínio: cria PENDING]
  └─ gateway.Authorize(...)                                [runtime: provider ativo ou roteado]
  └─ p.Authorize(ref) → p.Capture()  OU  p.Fail(reason)    [domínio aplica o resultado]
  └─ repo.Save(p)                                          [PERSISTÊNCIA por último]
  └─ publisher.Publish(eventos)                            [EventBus, após persistir]
```

A mutation despacha o command pelo **CommandBus** (Watermill, block-until-ack) e, ao
retornar, relê pelo read model (`GetPaymentByKey`) — como o id é gerado dentro do
use-case, a interface relê pela `IdempotencyKey` que ela mesma enviou.

A idempotência é garantida em três pontos: a `IdempotencyKey` (VO), o `Repository.Save` (rejeita chave duplicada), e o próprio gateway (mesma chave ⇒ mesma `GatewayReference`).

## Como rodar

Pré-requisito: Go 1.25+.

```bash
make generate    # gera o código do gqlgen (generated/, models_gen.go)
make tidy        # baixa dependências (gera go.sum)
make test        # testes de domínio (independem do código gerado)
make run         # sobe em http://localhost:8080
make dev         # desenvolvimento local com live reload (Air)
```

`make dev` usa o [Air](https://github.com/air-verse/air): roda `make generate` e
sobe o servidor recompilando a cada alteração em arquivos `.go` (config em
`.air.toml`). Usa o `air` global se estiver no `PATH`; senão, baixa via
`go run github.com/air-verse/air` (sem instalar nada). Editou o `schema.graphqls`?
Rode `make generate` — os `.go` regerados disparam o reload automaticamente.

O `make dev` define `APP_ENV=dev`, que liga o **logger de console colorido** (em
vez do JSON de produção) e silencia os eventos do framework fx para `debug`,
deixando os logs limpos. Em produção (sem `APP_ENV=dev`) o logger é JSON
estruturado. Dá para forçar em qualquer comando: `APP_ENV=dev make run`.

> O diretório `internal/payments/interfaces/graph/generated/` é **gerado** pelo gqlgen e não vem versionado. Rode `make generate` na primeira vez (e sempre que editar `schema.graphqls`).

Playground GraphQL: http://localhost:8080/ — endpoint: `POST /query`.

### Exemplo de mutation

```graphql
mutation {
  processPayment(input: {
    idempotencyKey: "order-2026-0001"
    customerId: "cust_42"
    amountCents: 4990
    currency: "BRL"
  }) {
    id status amountCents
  }
}
```

Reenviar a mesma `idempotencyKey` devolve o mesmo pagamento, sem cobrar de novo.

### Usuário + pedido (saga dispara o pagamento)

```graphql
# 1) cria um usuário
mutation { createUser(input: { name: "Marvin", email: "marvin@vaz.dev" }) { id } }

# 2) cria um pedido para esse cliente — o pagamento é processado automaticamente
mutation {
  createOrder(input: {
    idempotencyKey: "order-2026-0001"
    customerId: "usr_..."   # id do usuário criado acima
    amountCents: 9900
    currency: "BRL"
  }) { id status }
}

# queries de usuário
query { users { id name email } }
query { user(id: "usr_...") { name email } }
```

`createOrder` com um `customerId` inexistente é rejeitado (um pedido exige um cliente).

### Trocar o gateway em runtime

```bash
# ver provedores e o ativo
curl localhost:8080/admin/gateway

# trocar o provedor ativo sem reiniciar
curl -X POST localhost:8080/admin/gateway -H 'content-type: application/json' \
  -d '{"name":"stripe"}'
```

Também é possível rotear por requisição passando `gatewayName` no input da mutation.

## Federação

`Payment` é uma _entity_ federada via `@key(fields: "id")`. O `EntityResolver.FindPaymentByID` permite que outros subgraphs (ex.: um subgraph de `Customer`) referenciem `Payment` e que um Apollo Router / Cosmo monte o supergraph. O plugin de federation do gqlgen gera `_service` e `_entities` automaticamente.

## Notas de design

- **Money em centavos (int64)** evita erro de ponto flutuante; operações retornam novos VOs.
- **Concorrência otimista** via `Version` no `Update` do repositório.
- **Eventos de domínio** são acumulados no agregado e publicados só após persistir (pronto para virar um _outbox_).
- O `MemoryRepository` é para dev/teste; trocá-lo por Postgres é só implementar a porta `payment.Repository` e religar no `InfrastructureModule` — nada no domínio muda.
