package middleware

import (
	// "fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type Visitor struct {
	lastSeen time.Time
	requests int
}

type RateLimiter struct {
	visitors map[string]*Visitor
	mu sync.Mutex
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
	}

	go rl.cleanupVisitors()

	return rl
}

func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(1 * time.Minute)

		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Printf("Error parsing IP from RemoteAddr: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		rl.mu.Lock()

		v, exists := rl.visitors[ip]
		if !exists {
			rl.visitors[ip] = &Visitor{
				lastSeen: time.Now(),
				requests: 0,
			}
			v = rl.visitors[ip]
		}

		if time.Since(v.lastSeen) > 1*time.Second {
			v.requests = 0
		}

		v.lastSeen = time.Now()
		v.requests++

		if v.requests > 5 {
			// fmt.Printf("Rate limit exceeded for IP %s\n", ip)
			rl.mu.Unlock()
			http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
			return
		}

		rl.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

