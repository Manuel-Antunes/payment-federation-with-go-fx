// Package dataloader contém os DataLoaders (vektah/dataloaden) que fazem batch +
// cache POR REQUISIÇÃO das relações resolvidas no GraphQL, evitando N+1.
package dataloader

//go:generate go run github.com/vektah/dataloaden UserLoader string *github.com/example/payment-federation/internal/app/graph/model.User
//go:generate go run github.com/vektah/dataloaden OrderLoader string *github.com/example/payment-federation/internal/app/graph/model.Order
