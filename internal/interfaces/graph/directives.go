package graph

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/go-playground/validator/v10"
	nonstd "github.com/go-playground/validator/v10/non-standard/validators"

	"github.com/example/payment-federation/internal/interfaces/graph/generated"
)

// validate é a instância (concorrente-segura) do go-playground/validator usada
// pela diretiva. Registra "notblank" (do pacote non-standard) para rejeitar
// strings só com espaços — o `required` padrão considera " " preenchido.
var validate = func() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	_ = v.RegisterValidation("notblank", nonstd.NotBlank)
	return v
}()

// NewExecutableSchema monta o schema executável com os resolvers E as diretivas
// de validação registradas. Centralizar aqui mantém o servidor HTTP alheio aos
// detalhes do gqlgen/diretivas.
func NewExecutableSchema(r *Resolver) graphql.ExecutableSchema {
	return generated.NewExecutableSchema(generated.Config{
		Resolvers: r,
		Directives: generated.DirectiveRoot{
			Binding: binding,
		},
	})
}

// binding implementa a diretiva @binding sobre campos de input: aplica a tag do
// go-playground/validator (arg `constraint`) direto ao valor coagido, ANTES de
// qualquer command ser despachado. Um erro vira erro GraphQL no caminho do campo.
func binding(ctx context.Context, _ any, next graphql.Resolver, constraint string) (any, error) {
	val, err := next(ctx)
	if err != nil {
		return nil, err
	}

	// Campos opcionais ausentes (ponteiro nil) não são validados.
	target := deref(val)
	if target == nil || constraint == "" {
		return val, nil
	}

	if err := validate.Var(target, constraint); err != nil {
		var ves validator.ValidationErrors
		if errors.As(err, &ves) && len(ves) > 0 {
			return nil, errors.New(describe(ves[0]))
		}
		return nil, err
	}
	return val, nil
}

// describe traduz o erro do validator para uma mensagem amigável (pt-BR),
// distinguindo regras de comprimento (strings) das de valor (números).
func describe(fe validator.FieldError) string {
	switch fe.Tag() {
	case "notblank", "required":
		return "não pode ser vazio"
	case "min":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("deve ter no mínimo %s caractere(s)", fe.Param())
		}
		return fmt.Sprintf("deve ser >= %s", fe.Param())
	case "max":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("deve ter no máximo %s caractere(s)", fe.Param())
		}
		return fmt.Sprintf("deve ser <= %s", fe.Param())
	case "oneof":
		return "deve ser um de: " + strings.ReplaceAll(fe.Param(), " ", ", ")
	case "email":
		return "e-mail inválido"
	default:
		return fmt.Sprintf("valor inválido (regra %q)", fe.Tag())
	}
}

// deref remove um nível de ponteiro (devolvendo nil para ponteiro nil) para que
// o validator receba o valor concreto (string/int64/...).
func deref(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case *string:
		if t == nil {
			return nil
		}
		return *t
	case *int64:
		if t == nil {
			return nil
		}
		return *t
	case *int:
		if t == nil {
			return nil
		}
		return *t
	default:
		return v
	}
}
