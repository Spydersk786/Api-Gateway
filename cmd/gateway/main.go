package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"api-gateway/internal/config"
	"api-gateway/internal/middleware"
	"api-gateway/internal/health"
	"api-gateway/internal/loadbalancer"
)

func main() {
	mux := http.NewServeMux()
	
	var middlewareRegistry = map[string]middleware.Middleware{
		"Recover" : middleware.RecoverMiddleware,
		"Logging" : middleware.LoggingMiddleware,
		"RequestID" : middleware.RequestIDMiddleware,
		"RateLimit" : middleware.NewRateLimiter().Middleware, // Get called exactly once to create the rate limiter instance
		"Auth" : middleware.AuthMiddleware,
	}

	backend1 := &config.Backend{URL: "http://localhost:8081"}
	backend2 := &config.Backend{URL: "http://localhost:8082"}
	backend3 := &config.Backend{URL: "http://localhost:8083"}

	route1 := &config.Route{
		Middlewares: []string{"Recover", "Logging", "RequestID", "RateLimit", "Auth"},
		Backends: []*config.Backend{backend1,backend2,backend3},
	}
	
	cfg := &config.Config{
		ListenAddr: ":8080",
		Routes: map[string]*config.Route{
			"/users": route1,
		},
	}

	for _, route := range cfg.Routes {
		for _, backend := range route.Backends {
			targetURL, err := url.Parse(backend.URL)
			if err != nil {
				log.Fatalf("Invalid backend URL %s: %v", backend.URL, err)
			}

			backend.Proxy = httputil.NewSingleHostReverseProxy(targetURL)
			backend.SetAlive(true)
		}
	}

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Gateway is Alive"))
	})

	for path, route := range cfg.Routes {
		if len(route.Backends) == 0 {
			log.Fatalf("No backends configured for route %s", path)
		}

		lb := loadbalancer.NewRoundRobin(route.Backends)

		var handler http.Handler = lb

		for _, middlewareName := range route.Middlewares {
			if middleware, exists := middlewareRegistry[middlewareName]; exists {
				handler = middleware(handler)
			}
		}

		mux.Handle(path, handler)
	}

	go health.StartHealthCheck(route1.Backends)

	port := cfg.ListenAddr
	fmt.Printf("Starting API Gateway on port %s\n", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}

}