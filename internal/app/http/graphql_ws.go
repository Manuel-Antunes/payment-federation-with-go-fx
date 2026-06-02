package http

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gofiber/contrib/websocket"
	"go.uber.org/zap"

	"github.com/example/payment-federation/internal/app/graph/dataloader"
	"github.com/example/payment-federation/internal/shared/cqrs"
)

// graphql-transport-ws (https://github.com/enisdenjo/graphql-ws) implementado à
// mão sobre o WebSocket do Fiber (gofiber/contrib/websocket, base fasthttp),
// dirigindo o EXECUTOR do gqlgen diretamente.
//
// Por que à mão: o transporte WebSocket embutido do gqlgen usa gorilla/websocket
// + http.Hijacker (modelo net/http), incompatível com o hijack adiado do
// fasthttp. Então não dá pra "converter" — reimplementamos o protocolo e ligamos
// no mesmo executor que o transporte do gqlgen usaria por dentro.
//
// Tipos de mensagem (cliente->servidor): connection_init, subscribe, complete,
// ping, pong. (servidor->cliente): connection_ack, next, error, complete, pong.

type wsInbound struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// newGraphQLWSHandler devolve o handler de conexão do contrib/websocket. Cada
// conexão roda em sua própria goroutine; ao retornar, a conexão é fechada.
func newGraphQLWSHandler(exec graphql.GraphExecutor, queries *cqrs.QueryBus, log *zap.Logger) func(*websocket.Conn) {
	return func(conn *websocket.Conn) {
		c := &wsConn{
			conn:    conn,
			exec:    exec,
			queries: queries,
			log:     log.Named("graphql_ws"),
			out:     make(chan any, 32),
			done:    make(chan struct{}),
			ops:     make(map[string]context.CancelFunc),
		}
		c.serve()
	}
}

// wsConn carrega o estado de UMA conexão WebSocket. As escritas passam por um
// único writer (canal out) — *websocket.Conn não é seguro para writers concorrentes.
type wsConn struct {
	conn    *websocket.Conn
	exec    graphql.GraphExecutor
	queries *cqrs.QueryBus
	log     *zap.Logger

	out  chan any
	done chan struct{}
	once sync.Once

	mu  sync.Mutex
	ops map[string]context.CancelFunc
}

func (c *wsConn) serve() {
	// connCtx encerra todas as subscriptions de uma vez no disconnect.
	connCtx, cancelConn := context.WithCancel(context.Background())
	defer cancelConn()

	go c.writeLoop()
	defer c.shutdown()

	for {
		var msg wsInbound
		if err := c.conn.ReadJSON(&msg); err != nil {
			return // cliente desconectou / conexão quebrou
		}
		switch msg.Type {
		case "connection_init":
			c.send(map[string]any{"type": "connection_ack"})
		case "ping":
			c.send(map[string]any{"type": "pong"})
		case "pong":
			// keep-alive do cliente: ignora
		case "subscribe":
			c.startOperation(connCtx, msg)
		case "complete":
			c.cancelOperation(msg.ID)
		default:
			// tipo desconhecido: ignora (tolerante)
		}
	}
}

// startOperation cria a operação no executor do gqlgen e bombeia cada resposta
// como "next", terminando com "complete". Subscriptions geram N respostas;
// query/mutation geram uma só.
func (c *wsConn) startOperation(connCtx context.Context, msg wsInbound) {
	if msg.ID == "" {
		return
	}

	var params graphql.RawParams
	if err := json.Unmarshal(msg.Payload, &params); err != nil {
		c.send(map[string]any{"id": msg.ID, "type": "error",
			"payload": []map[string]any{{"message": err.Error()}}})
		return
	}
	now := graphql.Now()
	params.ReadTime = graphql.TraceTiming{Start: now, End: now}

	opCtx, cancel := context.WithCancel(connCtx)
	// Loaders por-operação (batch + cache no escopo da operação), como no HTTP.
	opCtx = dataloader.Attach(opCtx, dataloader.NewLoaders(c.queries))
	// O executor lê o start time do trace do contexto (o transporte HTTP do
	// gqlgen faz isto por baixo); sem ele, CreateOperationContext entra em panic.
	opCtx = graphql.StartOperationTrace(opCtx)

	rc, errl := c.exec.CreateOperationContext(opCtx, &params)
	if errl != nil {
		c.send(map[string]any{"id": msg.ID, "type": "error", "payload": errl})
		cancel()
		return
	}

	c.mu.Lock()
	c.ops[msg.ID] = cancel
	c.mu.Unlock()

	responses, execCtx := c.exec.DispatchOperation(opCtx, rc)
	go func() {
		defer c.cancelOperation(msg.ID)
		for {
			resp := responses(execCtx)
			if resp == nil {
				break
			}
			c.send(map[string]any{"id": msg.ID, "type": "next", "payload": resp})
		}
		c.send(map[string]any{"id": msg.ID, "type": "complete"})
	}()
}

func (c *wsConn) cancelOperation(id string) {
	c.mu.Lock()
	cancel, ok := c.ops[id]
	delete(c.ops, id)
	c.mu.Unlock()
	if ok {
		cancel()
	}
}

// writeLoop é o ÚNICO escritor da conexão.
func (c *wsConn) writeLoop() {
	for {
		select {
		case v := <-c.out:
			if err := c.conn.WriteJSON(v); err != nil {
				c.shutdown()
				return
			}
		case <-c.done:
			return
		}
	}
}

// send enfileira uma mensagem para o writer; no-op se a conexão já encerrou.
func (c *wsConn) send(v any) {
	select {
	case c.out <- v:
	case <-c.done:
	}
}

// shutdown sinaliza o fim da conexão (idempotente). O connCtx (cancelado no
// defer de serve) encerra as subscriptions; aqui só destravamos os sends.
func (c *wsConn) shutdown() {
	c.once.Do(func() { close(c.done) })
}
