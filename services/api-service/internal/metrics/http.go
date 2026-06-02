package metrics

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var HTTPRequests = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "asoc_http_requests_total",
		Help: "Total HTTP requests handled by api-service",
	},
	[]string{"method", "route", "code"},
)

// HTTPDuration — латентность ответа api-service (для Grafana loadtest, ingest и др.).
var HTTPDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "asoc_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60, 120, 180},
	},
	[]string{"method", "route", "code"},
)

// RouteLabel collapses paths to keep Prometheus cardinality low.
func RouteLabel(path string) string {
	switch path {
	case "/health", "/metrics", "/openapi.yaml":
		return path
	case "/api/v1/findings/ingest":
		return path
	}
	if strings.HasPrefix(path, "/api/v1/findings/") {
		return "/api/v1/findings/*"
	}
	if strings.HasPrefix(path, "/api/v1/console/") {
		return "/api/v1/console/*"
	}
	if strings.HasPrefix(path, "/api/v1/integrations/") {
		return "/api/v1/integrations/*"
	}
	if strings.HasPrefix(path, "/api/") {
		return "/api/*"
	}
	if strings.HasPrefix(path, "/swagger") {
		return "/swagger"
	}
	return "other"
}

func ObserveRequest(method, path string, status int, durationSec float64) {
	if status == 0 {
		status = 200
	}
	route := RouteLabel(path)
	code := strconv.Itoa(status)
	HTTPRequests.WithLabelValues(method, route, code).Inc()
	if durationSec >= 0 {
		HTTPDuration.WithLabelValues(method, route, code).Observe(durationSec)
	}
}
