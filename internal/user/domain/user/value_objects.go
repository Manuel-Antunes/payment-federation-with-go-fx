package user

import "strings"

// ---------------------------------------------------------------------------
// Value Objects do bounded context de Usuário. Imutáveis e auto-validáveis.
// ---------------------------------------------------------------------------

// UserID identifica unicamente um agregado User.
type UserID string

func (id UserID) String() string { return string(id) }
func (id UserID) IsZero() bool   { return id == "" }

// Email é um VO com validação mínima de formato. Normaliza para minúsculas.
type Email string

func NewEmail(raw string) (Email, error) {
	e := strings.ToLower(strings.TrimSpace(raw))
	if e == "" {
		return "", ErrEmptyEmail
	}
	at := strings.IndexByte(e, '@')
	// precisa de algo antes do @, um @, e um ponto no domínio depois do @.
	if at <= 0 || at == len(e)-1 || !strings.Contains(e[at+1:], ".") {
		return "", ErrInvalidEmail
	}
	return Email(e), nil
}

func (e Email) String() string { return string(e) }

// Name é o nome de exibição do usuário (não vazio).
type Name string

func NewName(raw string) (Name, error) {
	n := strings.TrimSpace(raw)
	if n == "" {
		return "", ErrEmptyName
	}
	return Name(n), nil
}

func (n Name) String() string { return string(n) }
