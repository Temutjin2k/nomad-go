package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"service", "method", "path", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "path", "status"},
	)

	HttpRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
		[]string{"service"},
	)

	// Business metrics
	RidesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rides_total",
			Help: "Total number of rides created",
		},
		[]string{"service", "status"},
	)

	DriversOnlineGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "drivers_online_total",
			Help: "Current number of online drivers",
		},
		[]string{"service"},
	)

	WebSocketConnectionsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "websocket_connections_total",
			Help: "Current number of active WebSocket connections",
		},
		[]string{"service"},
	)
)

// RecordHTTPMetrics records HTTP request metrics
func RecordHTTPMetrics(service, method, path string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	HttpRequestsTotal.WithLabelValues(service, method, path, status).Inc()
	HttpRequestDuration.WithLabelValues(service, method, path, status).Observe(duration.Seconds())
}
