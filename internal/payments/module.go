// Package payments é o módulo raiz do bounded context de PAGAMENTOS (que inclui
// o agregado Order). Une os módulos de aplicação e de infraestrutura num único
// fx.Module. Depende do kernel compartilhado (shared.Module) e, para validar o
// cliente de um pedido, da porta application.CustomerDirectory (provida na
// composição). A interface (GraphQL/HTTP) é montada no nível da aplicação.
package payments

import (
	"go.uber.org/fx"

	"github.com/example/payment-federation/internal/payments/application"
	"github.com/example/payment-federation/internal/payments/infrastructure"
)

// Module é o módulo do bounded context de pagamentos (aplicação + infraestrutura).
var Module = fx.Module("payments",
	application.Module,
	infrastructure.Module,
)
