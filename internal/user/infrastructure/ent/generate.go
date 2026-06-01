// Package ent é o cliente ent (entgo.io) do bounded context de USUÁRIO. A
// geração é por módulo: cada contexto tem o seu próprio schema e client,
// mantendo a fronteira de persistência. Id é string (gerado no domínio).
package ent

//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate ./schema
