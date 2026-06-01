package query

import (
	"context"

	"github.com/example/payment-federation/internal/user/domain/user"
)

// GetUsersByIDs é a query de LEITURA EM LOTE — resolve N usuários numa só
// chamada. É o que o DataLoader despacha para fazer o batch dos Order.customer.
type GetUsersByIDs struct{ IDs []string }

type GetUsersByIDsHandler struct{ repo user.Repository }

func NewGetUsersByIDsHandler(repo user.Repository) *GetUsersByIDsHandler {
	return &GetUsersByIDsHandler{repo: repo}
}

func (h *GetUsersByIDsHandler) Handle(ctx context.Context, q GetUsersByIDs) ([]*user.User, error) {
	ids := make([]user.UserID, 0, len(q.IDs))
	for _, id := range q.IDs {
		ids = append(ids, user.UserID(id))
	}
	return h.repo.FindByIDs(ctx, ids)
}
