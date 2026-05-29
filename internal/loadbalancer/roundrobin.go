package loadbalancer

import (
	"net/http"
	"sync/atomic"
	"api-gateway/internal/config"
)
type RoundRobin struct {
	backends []*config.Backend
	counter uint64
}

func NewRoundRobin(backends []*config.Backend) *RoundRobin {
	return &RoundRobin{
		backends: backends,
		// counter starts at 0 automatically
	}
}

func (rr *RoundRobin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	count := atomic.AddUint64(&rr.counter, 1)
	for i := 0; i < len(rr.backends); i++ {
		index := (count + uint64(i)) % uint64(len(rr.backends))
		if rr.backends[index].IsAlive() {
			rr.backends[index].Proxy.ServeHTTP(w, r)
			return
		}
	}

	http.Error(w, "502 Bad Gateway: All backend servers are down", http.StatusBadGateway)
}