package e2e

import "testing"

func TestUserMutationsAndQueries(t *testing.T) {
	e := newE2E(t)

	id := e.createUser("Ada Lovelace", "ada@example.com")

	// query por id
	var byID struct {
		User struct{ ID, Name, Email string } `json:"user"`
	}
	e.mustQuery(`query($id: ID!){ user(id:$id){ `+userFields+` } }`, map[string]any{"id": id}, &byID)
	if byID.User.Email != "ada@example.com" || byID.User.Name != "Ada Lovelace" {
		t.Fatalf("unexpected user: %+v", byID.User)
	}

	// listagem
	var list struct {
		Users []struct{ ID string } `json:"users"`
	}
	e.mustQuery(`query{ users{ id } }`, nil, &list)
	if len(list.Users) != 1 || list.Users[0].ID != id {
		t.Fatalf("expected exactly the created user in list, got %+v", list.Users)
	}

	// update (nome + email) e leitura refletindo a mudança
	var upd struct {
		UpdateUser struct{ ID, Name, Email string } `json:"updateUser"`
	}
	e.mustQuery(`mutation($in: UpdateUserInput!){ updateUser(input:$in){ `+userFields+` } }`,
		map[string]any{"in": map[string]any{"id": id, "name": "Ada B.", "email": "ada.b@example.com"}}, &upd)
	if upd.UpdateUser.Name != "Ada B." || upd.UpdateUser.Email != "ada.b@example.com" {
		t.Fatalf("update not reflected: %+v", upd.UpdateUser)
	}
}

func TestUserValidationAndIdempotency(t *testing.T) {
	e := newE2E(t)

	// e-mail inválido => erro de domínio até a borda.
	if msg := e.mustError(`mutation($in: CreateUserInput!){ createUser(input:$in){ id } }`,
		map[string]any{"in": map[string]any{"name": "X", "email": "not-an-email"}}); msg == "" {
		t.Fatal("expected error message for invalid email")
	}

	// nome vazio => erro.
	if msg := e.mustError(`mutation($in: CreateUserInput!){ createUser(input:$in){ id } }`,
		map[string]any{"in": map[string]any{"name": "  ", "email": "ok@example.com"}}); msg == "" {
		t.Fatal("expected error message for empty name")
	}

	// updateUser de id inexistente => erro.
	if msg := e.mustError(`mutation($in: UpdateUserInput!){ updateUser(input:$in){ id } }`,
		map[string]any{"in": map[string]any{"id": "usr_missing", "name": "Z"}}); msg == "" {
		t.Fatal("expected error for unknown user update")
	}

	// e-mail único: criar com o mesmo e-mail devolve o usuário existente
	// (read-after-write pela chave natural), sem duplicar.
	id1 := e.createUser("First", "dup@example.com")
	id2 := e.createUser("Second", "dup@example.com")
	if id1 != id2 {
		t.Fatalf("expected same user id for duplicate email, got %s and %s", id1, id2)
	}
	var list struct {
		Users []struct{ ID string } `json:"users"`
	}
	e.mustQuery(`query{ users{ id } }`, nil, &list)
	if len(list.Users) != 1 {
		t.Fatalf("duplicate email must not create a second user, got %d", len(list.Users))
	}
}
