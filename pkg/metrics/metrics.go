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
	ActiveRidesGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_rides_total",
			Help: "Current number of active rides",
		},
		[]string{"service"},
	)

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

	DatabaseQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"service", "operation", "status"},
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "operation"},
	)

	RabbitMQMessagesPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rabbitmq_messages_published_total",
			Help: "Total number of messages published to RabbitMQ",
		},
		[]string{"service", "queue", "status"},
	)

	RabbitMQMessagesConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rabbitmq_messages_consumed_total",
			Help: "Total number of messages consumed from RabbitMQ",
		},
		[]string{"service", "queue", "status"},
	)
)

// RecordHTTPMetrics records HTTP request metrics
func RecordHTTPMetrics(service, method, path string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	HttpRequestsTotal.WithLabelValues(service, method, path, status).Inc()
	HttpRequestDuration.WithLabelValues(service, method, path, status).Observe(duration.Seconds())
}

// RecordDatabaseQuery records database query metrics
func RecordDatabaseQuery(service, operation string, err error, duration time.Duration) {
	status := "success"
	if err != nil {
		status = "error"
	}
	DatabaseQueriesTotal.WithLabelValues(service, operation, status).Inc()
	DatabaseQueryDuration.WithLabelValues(service, operation).Observe(duration.Seconds())
}

// RecordRabbitMQPublish records RabbitMQ publish metrics
func RecordRabbitMQPublish(service, queue string, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	RabbitMQMessagesPublished.WithLabelValues(service, queue, status).Inc()
}

// RecordRabbitMQConsume records RabbitMQ consume metrics
func RecordRabbitMQConsume(service, queue string, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	RabbitMQMessagesConsumed.WithLabelValues(service, queue, status).Inc()
}
