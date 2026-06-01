package query

import (
	"context"
	"errors"

	"github.com/example/payment-federation/internal/user/domain/user"
)

// Exists indica se um usuário existe pelo id. Usado pelo módulo de pagamentos
// (via bridge) para validar o cliente de um pedido — sem acoplar os módulos.
type Exists struct{ UserID string }

type ExistsHandler struct{ repo user.Repository }

func NewExistsHandler(repo user.Repository) *ExistsHandler { return &ExistsHandler{repo: repo} }

func (h *ExistsHandler) Handle(ctx context.Context, q Exists) (bool, error) {
	_, err := h.repo.FindByID(ctx, user.UserID(q.UserID))
	if errors.Is(err, user.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
