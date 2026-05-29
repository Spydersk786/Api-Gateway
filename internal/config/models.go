package config

import (
	"net/http/httputil"
	"sync"
)
type Config struct {
	ListenAddr string
	Routes map[string]*Route
}

type Route struct {
	Middlewares []string
	Backends []*Backend
}

type Backend struct {
	URL string
	Proxy *httputil.ReverseProxy
	active bool
	mu sync.RWMutex
}

func (b *Backend) SetAlive(active bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.active = active
}

func (b *Backend) IsAlive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.active
}