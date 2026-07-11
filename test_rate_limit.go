package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	URL      = "http://localhost:8080/users"
	Requests = 20
)

func generateBearerToken() (string, error) {
	keyData, err := os.ReadFile("private.pem")
	if err != nil {
		return "", fmt.Errorf("read private key: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user-123",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return "Bearer " + tokenString, nil
}

func sendRequest(index int, bearerToken string) {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		fmt.Printf("request=%d error=create request: %v\n", index, err)
		return
	}

	req.Header.Set("Authorization", bearerToken)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("request=%d error=%v\n", index, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("request=%d status=%d error=read body: %v\n", index, resp.StatusCode, err)
		return
	}

	fmt.Printf("request=%d status=%d body=%s\n", index, resp.StatusCode, strings.TrimSpace(string(body)))
}

func main() {
	bearerToken, err := generateBearerToken()
    // bearerToken := "randomly-generated-token"
	if err != nil {
		fmt.Printf("failed to generate bearer token: %v\n", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup

	for i := 0; i < Requests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			sendRequest(index, bearerToken)
		}(i)
	}

	wg.Wait()
}
