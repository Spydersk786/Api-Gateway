package main

import (
	"fmt"
	"log"
	"net/http"
	"math/rand"
)

func main() {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if rand.Intn(10) < 7 { // Simulate a 30% chance of failure
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Backend is Down"))
			fmt.Printf("RequestID %v", r.Header.Get("X-Request-ID"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Backend is Alive"))
		fmt.Printf("RequestID %v", r.Header.Get("X-Request-ID"))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Backend is Alive"))
	})

	port := ":8083"
	fmt.Printf("Starting Backend on port %s\n", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Failed to start Backend: %v", err)
	}

}