package acme

import (
	"fmt"
	"net/http"
	"sync"
)

// HTTP01Provider provides HTTP-01 challenge handling
type HTTP01Provider struct {
	client     *Client
	challenges map[string]string
	mu         sync.RWMutex
}

// Present sets up the HTTP-01 challenge
func (p *HTTP01Provider) Present(domain, token, keyAuth string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.challenges == nil {
		p.challenges = make(map[string]string)
	}

	p.challenges[token] = keyAuth
	return nil
}

// CleanUp removes the HTTP-01 challenge
func (p *HTTP01Provider) CleanUp(domain, token, keyAuth string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.challenges, token)
	return nil
}

// ServeHTTP serves HTTP-01 challenge requests
func (p *HTTP01Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract token from path: /.well-known/acme-challenge/{token}
	if r.URL.Path[:len("/.well-known/acme-challenge/")] != "/.well-known/acme-challenge/" {
		http.NotFound(w, r)
		return
	}

	token := r.URL.Path[len("/.well-known/acme-challenge/"):]
	if token == "" {
		http.NotFound(w, r)
		return
	}

	p.mu.RLock()
	keyAuth, exists := p.challenges[token]
	p.mu.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, keyAuth)
}

// GetChallengeResponse returns the challenge response for a token
func (p *HTTP01Provider) GetChallengeResponse(token string) (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	keyAuth, exists := p.challenges[token]
	return keyAuth, exists
}
