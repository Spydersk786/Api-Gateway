package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Define the Metrics
var (
	// Rate & Errors: Count total number of requests labeled by method, route, and status code
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_http_requests_total",
			Help: "Total number of HTTP requests processed by the Gateway",
		},
		[]string{"method", "path", "status_code"},
	)

	// Duration: Track how long requests take to process
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:	"gateway_http_request_duration_seconds",
			Help:   "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets, // Default buckets provided by Prometheus
		},
		[]string{"method", "path"},
	)
)

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(sw, r)
		duration := time.Since(start).Seconds()
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(sw.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}