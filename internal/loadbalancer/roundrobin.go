package laodbalancer

import (
	"api-gateway/internal/config"
)
type RoundRobin struct {
	backends []*config.Backend
	counter uint64
}