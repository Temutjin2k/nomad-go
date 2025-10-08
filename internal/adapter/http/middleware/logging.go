package middleware

import (
	"net/http"
	"time"
)

// Logging injects a request ID into the context and logs the request details.
func (a *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriterWrapper{
			ResponseWriter: w,
		}

		// 1. Log request start
		a.log.Debug(
			r.Context(),
			"started",
			"method", r.Method,
			"URL", r.URL.Path,
			"request-host", r.Host,
		)

		// 2. Serve the request
		next.ServeHTTP(rw, r)

		// 3. Log request end
		duration := time.Since(start)
		a.log.Debug(
			r.Context(),
			"completed",
			"method", r.Method,
			"URL", r.URL.Path,
			"status", rw.status,
			"duration", duration,
		)
	})
}

// responseWriterWrapper wraps http.ResponseWriter to track response status
type responseWriterWrapper struct {
	http.ResponseWriter
	status int
}

// WriteHeader intercepts the status code before writing headers
func (rw *responseWriterWrapper) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Write implements the http.ResponseWriter interface
func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	// If status wasn't set explicitly, default to 200 OK
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}
