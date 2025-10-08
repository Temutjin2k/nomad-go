package middleware

import (
	"fmt"
	"net/http"
)

func (app *Middleware) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if panic := recover(); panic != nil {
				w.Header().Set("Connection", "close")
				errorResponse(w, http.StatusInternalServerError, fmt.Errorf("%s", panic))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
