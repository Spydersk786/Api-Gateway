package middleware

import (
	"fmt"
	"net/http"
	"github.com/google/uuid"
)

type Middleware func(http.Handler) http.Handler

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// May be client sends their own id, need to handle that
		if r.Header.Get("X-Request-ID") == "" {
			requestID := uuid.NewString()
			fmt.Printf("Generated Request ID: %s\n", requestID)
			r.Header.Set("X-Request-ID", requestID)
		}
		next.ServeHTTP(w, r)
	})
}