package tokens

import (
	"fmt"
	"sync"
)

// Pool manages token rotation for multiple providers
type Pool struct {
	mu       sync.Mutex
	providers map[string]*ProviderPool // domain -> provider pool
}

// ProviderPool manages tokens for a single provider
type ProviderPool struct {
	provider *Provider
	tokens   []string
	index    int
}

// NewPool creates a new token pool
func NewPool() *Pool {
	return &Pool{
		providers: make(map[string]*ProviderPool),
	}
}

// AddProvider adds a provider with its tokens to the pool
func (p *Pool) AddProvider(provider *Provider, tokens []string) error {
	if len(tokens) == 0 {
		return fmt.Errorf("no tokens provided for %s", provider.Name)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.providers[provider.Domain] = &ProviderPool{
		provider: provider,
		tokens:   tokens,
		index:    0,
	}

	return nil
}

// GetToken returns the next token for a given domain using round-robin
func (p *Pool) GetToken(domain string) (string, *Provider, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	providerPool, exists := p.providers[domain]
	if !exists {
		return "", nil, fmt.Errorf("no tokens available for domain: %s", domain)
	}

	// Get current token
	token := providerPool.tokens[providerPool.index]

	// Advance to next token (round-robin)
	providerPool.index = (providerPool.index + 1) % len(providerPool.tokens)

	return token, providerPool.provider, nil
}

// HasTokens returns true if the pool has tokens for the given domain
func (p *Pool) HasTokens(domain string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, exists := p.providers[domain]
	return exists
}

// ProviderCount returns the number of providers with tokens
func (p *Pool) ProviderCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.providers)
}

// TokenCount returns the total number of tokens across all providers
func (p *Pool) TokenCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	count := 0
	for _, pp := range p.providers {
		count += len(pp.tokens)
	}
	return count
}

// Providers returns a list of provider names with tokens
func (p *Pool) Providers() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	var names []string
	for _, pp := range p.providers {
		names = append(names, pp.provider.Name)
	}
	return names
}
