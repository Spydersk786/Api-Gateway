package health

import (
	"net/http"
	"log"
	"time"

	"api-gateway/internal/config"
)

func StartHealthCheck(backends []*config.Backend) {

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	ticker := time.NewTicker(10 * time.Second)

	for {
		<-ticker.C

		for _, b := range backends {
			healthURL := b.URL + "/health"
			resp, err := client.Get(healthURL)

			if err != nil {
				log.Printf("Health check failed for backend %s: %v", b.URL, err)
				b.SetAlive(false)
				continue
			}

			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				if !b.IsAlive() {
					log.Printf("Backend %s is now healthy", b.URL)
				}
				b.SetAlive(true)
			}else {
				log.Printf("Health check returned non-OK for backend %s: %d", b.URL, resp.StatusCode)
				b.SetAlive(false)
			}
		}
	}
}


