package main

import (
	"api-gateway/internal/config"
	"api-gateway/internal/health"
	"api-gateway/internal/loadbalancer"
	"api-gateway/internal/middleware"
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr string `yaml:"listen_addr"`
	Routes map[string]Route `yaml:"routes"`
}

type Route struct {
	Middlewares []string `yaml:"middlewares"`
	Backends []string `yaml:"backends"`
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func main() {
	yamlConfig, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Default Redis address if not set in environment
	}

	redisLimiter := middleware.NewRateLimiter(redisAddr, 5, 10) // Example: 5 requests per second with a capacity of 10 tokens

	mux := http.NewServeMux()
	
	var middlewareRegistry = map[string]middleware.Middleware{
		"Recover" : middleware.RecoverMiddleware,
		"Logging" : middleware.LoggingMiddleware,
		"RequestID" : middleware.RequestIDMiddleware,
		"RateLimit" : redisLimiter.Middleware, // Get called exactly once to create the rate limiter instance
		"Auth" : middleware.AuthMiddleware,
	}

	allBackends := make(map[string]*config.Backend)

	for _, route := range yamlConfig.Routes {
		for _, backendURL := range route.Backends {
			if _, exists := allBackends[backendURL]; !exists {
				targetURL, err := url.Parse(backendURL)
				if err != nil {
					log.Fatalf("Invalid backend URL %s: %v", backendURL, err)
				}
				allBackends[backendURL] = &config.Backend{
					URL: backendURL,
					Proxy: httputil.NewSingleHostReverseProxy(targetURL),
				}
				allBackends[backendURL].SetAlive(true)
			}
		}
	}

	allRoutes := make(map[string]*config.Route)

	for path, routeConfig := range yamlConfig.Routes {
		route := &config.Route{
			Middlewares: routeConfig.Middlewares,
			Backends: []*config.Backend{},
		}
		for _, backendURL := range routeConfig.Backends {
			if backend, exists := allBackends[backendURL]; exists {
				route.Backends = append(route.Backends, backend)
			} else {
				log.Fatalf("Backend URL %s for route %s not found in backends map", backendURL, path)
			}
		}
		allRoutes[path] = route
	}
	
	cfg := &config.Config{
		ListenAddr: yamlConfig.ListenAddr,
		Routes: allRoutes,
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

	for _, route := range cfg.Routes {
		go health.StartHealthCheck(route.Backends)
	}

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
	}

	go func() {
		log.Printf("Starting API Gateway on port %s\n", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start API Gateway: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)

	// Listen for interrupt signals to gracefully shutdown the server
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive a signal
	<-quit
	log.Println("Shutting down API Gateway...")

	// Create a context with timeout to allow for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to gracefully shutdown API Gateway: %v", err)
	}

	log.Println("API Gateway stopped. All connections closed.")
}