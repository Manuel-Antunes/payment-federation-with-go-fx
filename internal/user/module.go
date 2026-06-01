// Package user é o módulo raiz do bounded context de USUÁRIO. Une os módulos
// de aplicação e de infraestrutura num único fx.Module. Depende do kernel
// compartilhado (shared.Module) para relógio e mensageria.
package user

import (
	"go.uber.org/fx"

	"github.com/example/payment-federation/internal/user/application"
	"github.com/example/payment-federation/internal/user/infrastructure"
)

// Module é o módulo do bounded context de usuário (aplicação + infraestrutura).
var Module = fx.Module("user",
	application.Module,
	infrastructure.Module,
)
