package cqrs_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"

	"github.com/example/payment-federation/internal/shared/cqrs"
	"github.com/example/payment-federation/internal/shared/messaging"
)

// --- Command de teste + handler que RETORNA um resultado ---

type greet struct {
	Name string
}

type greetResult struct {
	Message string
}

type greetHandler struct {
	err error // se != nil, o handler falha (testa propagação de erro)
}

func (h greetHandler) Handle(ctx context.Context, cmd greet) (greetResult, error) {
	if h.err != nil {
		return greetResult{}, h.err
	}
	return greetResult{Message: "hello " + cmd.Name}, nil
}

// newCommandBus monta o stack real do Watermill (GoChannel + router + processor)
// e devolve o nosso CommandBus já com o router rodando.
func newCommandBus(t *testing.T, handler cqrs.CommandHandler[greet, greetResult]) *cqrs.CommandBus {
	t.Helper()
	logger := watermill.NopLogger{}

	pubsub := messaging.NewPubSub(logger)
	router, err := messaging.NewRouter(logger)
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	transport, err := messaging.NewCommandBus(pubsub, logger)
	if err != nil {
		t.Fatalf("transport: %v", err)
	}
	processor, err := messaging.NewCommandProcessor(router, pubsub, logger)
	if err != nil {
		t.Fatalf("processor: %v", err)
	}

	bus := cqrs.NewCommandBus(transport, processor)
	if err := cqrs.RegisterCommand(bus, "greet", handler); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Handlers já registrados no processor -> sobe o router.
	go func() { _ = router.Run(context.Background()) }()
	select {
	case <-router.Running():
	case <-time.After(2 * time.Second):
		t.Fatal("router não subiu a tempo")
	}
	t.Cleanup(func() { _ = router.Close() })

	return bus
}

func TestExecuteMutationSync_ReturnsResult(t *testing.T) {
	bus := newCommandBus(t, greetHandler{})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := cqrs.ExecuteMutationSync[greet, greetResult](ctx, bus, greet{Name: "marvin"})
	if err != nil {
		t.Fatalf("ExecuteMutationSync: %v", err)
	}
	if res.Message != "hello marvin" {
		t.Fatalf("resultado inesperado: %q", res.Message)
	}
}

func TestExecuteMutationSync_PropagatesError(t *testing.T) {
	wantErr := errors.New("boom")
	bus := newCommandBus(t, greetHandler{err: wantErr})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := cqrs.ExecuteMutationSync[greet, greetResult](ctx, bus, greet{Name: "x"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("esperava erro %v, veio %v", wantErr, err)
	}
}

func TestDispatch_FireAndForget(t *testing.T) {
	bus := newCommandBus(t, greetHandler{})

	// Dispatch não retorna dado; só não pode falhar no despacho.
	if err := bus.Dispatch(context.Background(), greet{Name: "y"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
}
