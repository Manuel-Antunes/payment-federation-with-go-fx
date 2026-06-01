package user

import "errors"

// Erros de domínio puros — traduzíveis por qualquer camada (GraphQL, HTTP...).
var (
	ErrEmptyName    = errors.New("user: nome é obrigatório")
	ErrEmptyEmail   = errors.New("user: e-mail é obrigatório")
	ErrInvalidEmail = errors.New("user: e-mail inválido")

	// Persistência (porta)
	ErrNotFound   = errors.New("user: não encontrado")
	ErrEmailTaken = errors.New("user: já existe um usuário com este e-mail")
)
