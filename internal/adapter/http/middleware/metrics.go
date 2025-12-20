package middleware

import (
	"bufio"
	"net"
	"net/http"
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/metrics"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// Hijack allows the responseWriter to implement the http.Hijacker interface
// It was a problem when using WebSockets with the metrics middleware in gorrilla/websocket
// because the websocket.Upgrader requires the ResponseWriter to implement http.Hijacker.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return h.Hijack()
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Metrics middleware records HTTP metrics
func (m *Middleware) Metrics(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip metrics endpoint to avoid recursion
			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			metrics.HttpRequestsInFlight.WithLabelValues(serviceName).Inc()
			defer metrics.HttpRequestsInFlight.WithLabelValues(serviceName).Dec()

			// Wrap response writer to capture status code
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // default status
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			metrics.RecordHTTPMetrics(serviceName, r.Method, r.URL.Path, rw.statusCode, duration)
		})
	}
}
