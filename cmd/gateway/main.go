package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"api-gateway/internal/config"
	"api-gateway/internal/middleware"
)

var middlewareRegistry = map[string]middleware.Middleware{
	"RequestID" : middleware.RequestIDMiddleware,
}

func main() {
	mux := http.NewServeMux()
	
	backend1 := &config.Backend{URL: "http://localhost:8081"}
	
	route1 := &config.Route{
		Middlewares: []string{"RequestID"},
		Backends: []*config.Backend{backend1},
	}
	cfg := &config.Config{
		ListenAddr: ":8080",
		Routes: map[string]*config.Route{
			"/users": route1,
		},
	}
	
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Gateway is ALive"))
	})

	for path, route := range cfg.Routes {
		if len(route.Backends) == 0 {
			log.Fatalf("No backends configured for route %s", path)
		}

		targetStr := route.Backends[0].URL

		targetURL, err := url.Parse(targetStr)
		if err != nil {
			log.Fatalf("Invalid backend URL %s for route %s: %v", targetStr, path, err)
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		var handler http.Handler = proxy
		for _, middlewareName := range route.Middlewares {
			if middleware, exists := middlewareRegistry[middlewareName]; exists {
				handler = middleware(handler)
			}
		}

		mux.Handle(path, handler)

		log.Printf("Route %s configured to proxy to %s", path, targetStr)
	}

	port := ":8080"
	fmt.Printf("Starting API Gateway on port %s\n", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}

}