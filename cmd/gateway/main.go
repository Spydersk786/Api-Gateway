package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
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
}

func main() {
	mux := http.NewServeMux()
	
	
	backend1 := &Backend{URL: "http://localhost:8081"}
	
	route1 := &Route{
		Middlewares: []string{"AuthMiddleware", "LoggingMiddleware"},
		Backends: []*Backend{backend1},
	}
	cfg := &Config{
		ListenAddr: ":8080",
		Routes: map[string]*Route{
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

		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			proxy.ServeHTTP(w, r)
		})

		log.Printf("Route %s configured to proxy to %s", path, targetStr)
	}

	port := ":8080"
	fmt.Printf("Starting API Gateway on port %s\n", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}

}