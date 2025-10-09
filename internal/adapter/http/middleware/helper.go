package middleware

import (
	"encoding/json"
	"errors"
	"maps"
	"net/http"
)

// envelope is an alias for a map used to wrap JSON responses.
type envelope map[string]any

// errorResponse is a helper method for sending JSON-formatted error responses.
func errorResponse(w http.ResponseWriter, status int, message any) {
	env := envelope{"error": message}

	// Write the response using the writeJSON() helper. If this happens to return an
	// error then log it, and fall back to sending the client an empty response with a
	// 500 Internal Server Error status code.
	if err := writeJSON(w, status, env, nil); err != nil {
		w.WriteHeader(500)
	}
}

// writeJSON writes the given data as JSON into the response body.
func writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.Marshal(data)
	if err != nil {
		return errors.New("failed to encode json")
	}

	maps.Copy(w.Header(), headers)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}
