package loadbalancer

import (
	"log"
	"net/http"
	"math"
	"math/rand"
	"time"
	"sync/atomic"
	"context"
	"api-gateway/internal/config"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

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
	maxRetries := 3
	for i := 0; i < len(rr.backends); i++ {
		index := (count + uint64(i)) % uint64(len(rr.backends))
		backend := rr.backends[index]

		if !backend.IsAlive() {
			continue
		}

		for attempts := 0; attempts < maxRetries; attempts++ {
			recorder := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default to 200 OK
			}
			
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)

			reqWithTimeout := r.WithContext(ctx) // Ensure we have a context to work with
	
			backend.Proxy.ServeHTTP(recorder, reqWithTimeout)

			cancel() // Cancel the context to free resources

			if recorder.statusCode < 500 {
				log.Printf("RequestID %v served by backend in %v tries\n on backend %v", r.Header.Get("X-Request-ID"), attempts+1, backend.URL)
				for key, values := range recorder.Header() {
					w.Header()[key] = values
				}

				w.WriteHeader(recorder.statusCode)
				w.Write(recorder.body)
				return
			}

			if attempts < maxRetries-1 {
				baseDelay := 100 * time.Millisecond
				backoff := time.Duration(math.Pow(2, float64(attempts))) * baseDelay
				jitter := time.Duration(rand.Intn(50)) * time.Millisecond
				time.Sleep(time.Duration(backoff) + jitter)
			}
		}
		backend.SetAlive(false)
	}
	http.Error(w, "502 Bad Gateway: All backend servers are down", http.StatusBadGateway)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return len(b), nil
}