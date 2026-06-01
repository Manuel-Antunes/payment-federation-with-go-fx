package dataloader

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/example/payment-federation/internal/interfaces/graph/model"
	pquery "github.com/example/payment-federation/internal/payments/application/query"
	"github.com/example/payment-federation/internal/shared/cqrs"
	uquery "github.com/example/payment-federation/internal/user/application/query"
)

// Loaders agrupa os DataLoaders de UMA requisição. Como são criados por request
// (ver Middleware), o cache do loader é por-requisição — sem vazar dados entre
// requisições nem servir leitura obsoleta.
type Loaders struct {
	UserByID  *UserLoader
	OrderByID *OrderLoader
}

// Middleware é a função de middleware HTTP que injeta os loaders. Tipo nomeado
// para o fx resolver sem ambiguidade.
type Middleware func(http.Handler) http.Handler

type ctxKey struct{}

// NewMiddleware devolve um middleware que, a CADA requisição, cria um conjunto
// fresco de loaders (cache vazio) e o coloca no contexto.
func NewMiddleware(queries *cqrs.QueryBus) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			loaders := &Loaders{
				UserByID: NewUserLoader(UserLoaderConfig{
					Wait:     2 * time.Millisecond,
					MaxBatch: 100,
					Fetch:    fetchUsers(queries),
				}),
				OrderByID: NewOrderLoader(OrderLoaderConfig{
					Wait:     2 * time.Millisecond,
					MaxBatch: 100,
					Fetch:    fetchOrders(queries),
				}),
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, loaders)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// For devolve os loaders da requisição corrente (nil se o middleware não rodou).
func For(ctx context.Context) *Loaders {
	l, _ := ctx.Value(ctxKey{}).(*Loaders)
	return l
}

// fetchUsers é a função de BATCH do loader: recebe N ids e devolve N usuários
// na MESMA ordem das chaves (erro por chave ausente). Despacha UMA query em
// lote no QueryBus — o ganho do DataLoader sobre N buscas individuais.
func fetchUsers(queries *cqrs.QueryBus) func(ids []string) ([]*model.User, []error) {
	return func(ids []string) ([]*model.User, []error) {
		users := make([]*model.User, len(ids))
		errs := make([]error, len(ids))

		views, err := cqrs.Ask[uquery.GetUsersByIDs, []uquery.UserView](
			context.Background(), queries, uquery.GetUsersByIDs{IDs: ids},
		)
		if err != nil {
			for i := range errs {
				errs[i] = err
			}
			return users, errs
		}

		byID := make(map[string]uquery.UserView, len(views))
		for _, v := range views {
			byID[v.ID] = v
		}
		for i, id := range ids {
			if v, ok := byID[id]; ok {
				users[i] = &model.User{ID: v.ID, Name: v.Name, Email: v.Email}
			} else {
				errs[i] = fmt.Errorf("usuário %q não encontrado", id)
			}
		}
		return users, errs
	}
}

// fetchOrders é a função de BATCH do loader de pedidos: N ids -> N pedidos na
// ordem das chaves (erro por chave ausente), via UMA query GetOrdersByIDs.
func fetchOrders(queries *cqrs.QueryBus) func(ids []string) ([]*model.Order, []error) {
	return func(ids []string) ([]*model.Order, []error) {
		orders := make([]*model.Order, len(ids))
		errs := make([]error, len(ids))

		views, err := cqrs.Ask[pquery.GetOrdersByIDs, []pquery.OrderView](
			context.Background(), queries, pquery.GetOrdersByIDs{IDs: ids},
		)
		if err != nil {
			for i := range errs {
				errs[i] = err
			}
			return orders, errs
		}

		byID := make(map[string]pquery.OrderView, len(views))
		for _, v := range views {
			byID[v.ID] = v
		}
		for i, id := range ids {
			if v, ok := byID[id]; ok {
				orders[i] = &model.Order{
					ID:          v.ID,
					CustomerID:  v.CustomerID,
					AmountCents: v.AmountCents,
					Currency:    v.Currency,
					Status:      v.Status,
				}
			} else {
				errs[i] = fmt.Errorf("pedido %q não encontrado", id)
			}
		}
		return orders, errs
	}
}
