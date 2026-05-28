package config

import (
	"net/http/httputil"
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
	Proxy *httputil.ReverseProxy
}