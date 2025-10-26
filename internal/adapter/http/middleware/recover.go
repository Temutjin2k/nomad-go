package middleware

import (
	"fmt"
	"net/http"
)

func (a *Middleware) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if panic := recover(); panic != nil {
				a.log.Error(r.Context(), "panic occured in http server", fmt.Errorf("%s", panic))
				w.Header().Set("Connection", "close")
				errorResponse(w, http.StatusInternalServerError, "the server encountered a problem and could not process your request")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
