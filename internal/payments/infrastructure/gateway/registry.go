package gateway

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/example/payment-federation/internal/payments/domain/payment"
)

// Registry implementa payment.GatewayProvider permitindo TROCA EM RUNTIME do
// provedor ativo sem reiniciar o serviço.
//
//   - Register adiciona/atualiza um gateway disponível.
//   - SetActive troca atomicamente o provedor padrão.
//   - Active/Resolve são lock-free na leitura (atomic.Pointer + RWMutex no mapa).
type Registry struct {
	mu       sync.RWMutex
	gateways map[string]payment.Gateway
	active   atomic.Pointer[payment.Gateway]
}

var _ payment.GatewayProvider = (*Registry)(nil)

func NewRegistry() *Registry {
	return &Registry{gateways: make(map[string]payment.Gateway)}
}

// Register adiciona um gateway. Se for o primeiro, vira o ativo.
func (r *Registry) Register(gw payment.Gateway) {
	r.mu.Lock()
	r.gateways[gw.Name()] = gw
	first := len(r.gateways) == 1
	r.mu.Unlock()
	if first {
		g := gw
		r.active.Store(&g)
	}
}

// SetActive troca o gateway padrão em runtime. Retorna erro se não registrado.
func (r *Registry) SetActive(name string) error {
	r.mu.RLock()
	gw, ok := r.gateways[name]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("gateway %q não registrado", name)
	}
	r.active.Store(&gw)
	return nil
}

// Active devolve o gateway ativo (leitura atômica, sem lock).
func (r *Registry) Active() payment.Gateway {
	if g := r.active.Load(); g != nil {
		return *g
	}
	return nil
}

// Resolve devolve um gateway por nome (roteamento específico).
func (r *Registry) Resolve(name string) (payment.Gateway, error) {
	if name == "" {
		return nil, fmt.Errorf("nome de gateway vazio")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	gw, ok := r.gateways[name]
	if !ok {
		return nil, fmt.Errorf("gateway %q não encontrado", name)
	}
	return gw, nil
}

// Available lista os gateways registrados (para observabilidade/admin).
func (r *Registry) Available() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.gateways))
	for n := range r.gateways {
		names = append(names, n)
	}
	return names
}
