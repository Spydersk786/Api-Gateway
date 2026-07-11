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
	BaseURL  = "http://localhost:8080"
	Requests = 20
)

type requestSpec struct {
	method  string
	path    string
	body    string
}

func generateBearerToken() (string, error) {
	keyPaths := []string{"internal/keys/private.pem", "private.pem"}

	var lastErr error
	for _, keyPath := range keyPaths {
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			lastErr = err
			continue
		}

		privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
		if err != nil {
			lastErr = err
			continue
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

	if lastErr != nil {
		return "", fmt.Errorf("read private key: %w", lastErr)
	}

	return "", fmt.Errorf("private key not found")
}

func requestPlan() []requestSpec {
	return []requestSpec{
		{method: http.MethodGet, path: "/users"},
		{method: http.MethodPost, path: "/users", body: `{"name":"alice"}`},
		{method: http.MethodPut, path: "/users", body: `{"name":"alice-updated"}`},
		{method: http.MethodDelete, path: "/users", body: `{"name":"alice-delete"}`},
		{method: http.MethodGet, path: "/billings"},
		{method: http.MethodPost, path: "/billings", body: `{"amount":42}`},
		{method: http.MethodPut, path: "/billings", body: `{"amount":99}`},
		{method: http.MethodDelete, path: "/billings", body: `{"amount":0}`},
	}
}

func sendRequest(index int, spec requestSpec, bearerToken string) {
	client := &http.Client{Timeout: 5 * time.Second}

	bodyReader := strings.NewReader(spec.body)
	req, err := http.NewRequest(spec.method, BaseURL+spec.path, bodyReader)
	if err != nil {
		fmt.Printf("request=%d error=create request: %v\n", index, err)
		return
	}

	req.Header.Set("Authorization", bearerToken)
	if spec.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("request=%d method=%s path=%s error=%v\n", index, spec.method, spec.path, err)
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("request=%d method=%s path=%s status=%d error=read body: %v\n", index, spec.method, spec.path, resp.StatusCode, err)
		return
	}

	fmt.Printf("request=%d method=%s path=%s status=%d body=%s\n", index, spec.method, spec.path, resp.StatusCode, strings.TrimSpace(string(responseBody)))
}

func main() {
	bearerToken, err := generateBearerToken()
	if err != nil {
		fmt.Printf("failed to generate bearer token: %v\n", err)
		os.Exit(1)
	}

	plans := requestPlan()
	var wg sync.WaitGroup

	for i := 0; i < Requests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			spec := plans[index%len(plans)]
			sendRequest(index, spec, bearerToken)
		}(i)
	}

	wg.Wait()
}
