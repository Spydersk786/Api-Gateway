package main

import (
	"io"
	"fmt"
	"log"
	// "time"
	"net/http"
	"math/rand"
)

func main() {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		// simulate request taking more than 2 second to trigger context timeout
		// time.Sleep(10 * time.Second)
		if rand.Intn(10) < 2 { // Simulate a 20% chance of failure
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Backend is Down"))
			fmt.Printf("RequestID %v failed", r.Header.Get("X-Request-ID"))
			return
		}

		if r.Method == http.MethodPost{
			body, _ := io.ReadAll(r.Body)
			defer r.Body.Close()

			fmt.Printf("Received POST request with body: %s\n", string(body))

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("Gateway successfully forwarded POST request to Backend"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Backend is Alive"))
		fmt.Printf("RequestID %v succeeded", r.Header.Get("X-Request-ID"))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Backend is Alive"))
	})

	port := ":8081"
	fmt.Printf("Starting Backend on port %s\n", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Failed to start Backend: %v", err)
	}

}