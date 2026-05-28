package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Backend is ALive"))
	})

	port := ":8081"
	fmt.Printf("Starting Backend on port %s\n", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Failed to start Backend: %v", err)
	}

}