package middleware

import (
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"github.com/golang-jwt/jwt/v5"
)

type JWTValidator struct {
	publicKey *rsa.PublicKey
}

func NewJWTValidator(publicKeyPath string) (*JWTValidator) {
	keyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		log.Fatalf("[JWT] Failed to read public key from %s: %v", publicKeyPath, err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		log.Fatalf("[JWT] Failed to parse public key: %v", err)
	}

	log.Printf("[JWT] Successfully loaded public key for Authentication")

	return &JWTValidator{
		publicKey: publicKey,
	}
}

func (j *JWTValidator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "401 Unauthorized: Missing Authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")

		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "401 Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// as no key rotation is implemented, we always return the same public key for verification
			return j.publicKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "401 Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

