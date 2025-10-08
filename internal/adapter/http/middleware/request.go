package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
)

// RequestID injects request_id to the request ctx
func (a *Middleware) RequestID(next http.Handler) http.Handler {
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
