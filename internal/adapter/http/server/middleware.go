package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
)

func (a *API) withMiddleware() http.Handler {
	return a.RequestIDMiddleware(
		a.mux,
	)
}

// RequestLoggingMiddleware injects a request ID into the context and logs the request details.
func (a *API) RequestLoggingMiddleware(next http.Handler) http.Handler {
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

// RequestIDMiddleware injects request_id to the request ctx
func (a *API) RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Reuse incoming X-Request-ID if provided
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			// 2. Otherwise generate one
			reqID = newRequestID()
		}

		// 3. Echo to clients for debugging / tracing
		w.Header().Set("X-Request-ID", reqID)

		// 4. Inject into context for our logger
		ctx := wrap.WithRequestID(r.Context(), reqID)

		// 5. Call the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// newRequestID returns a 16-byte random hex string, e.g. “9f86d081884c7d65…”
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback to timestamp if crypto/rand fails
		return hex.EncodeToString(fmt.Appendf(nil, "%d", time.Now().UnixNano()))
	}
	return hex.EncodeToString(b)
}
