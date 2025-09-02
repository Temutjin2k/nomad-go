package handler

import "net/http"

func errorResponse(w http.ResponseWriter, status int, message any) {
	env := envelope{"error": message}

	// Write the response using the writeJSON() helper. If this happens to return an
	// error then log it, and fall back to sending the client an empty response with a
	// 500 Internal Server Error status code.
	if err := writeJSON(w, status, env, nil); err != nil {
		w.WriteHeader(500)
	}
}

// failedValidationResponse returns 422 UnprocessableEntity status.
// Why we choose 422 status?
// The HTTP 422 Unprocessable Content client error response status code indicates
// that the server understood the content type of the request content, and the
// syntax of the request content was correct, but it was unable to process the
// contained instructions.
// Clients that receive a 422 response should expect that repeating the request
// without modification will fail with the same error.
func failedValidationResponse(w http.ResponseWriter, errors map[string]string) {
	errorResponse(w, http.StatusUnprocessableEntity, errors)
}

// badRequestResponse returns 400 BadRequest status
// The HTTP 400 Bad Request client error response status code indicates that
// the server would not process the request due to something the server considered
// to be a client error. The reason for a 400 response is typically due to malformed
// request syntax, invalid request message framing, or deceptive request routing.
func badRequestResponse(w http.ResponseWriter, message any) {
	errorResponse(w, http.StatusUnprocessableEntity, message)
}

// internalErrorResponse returns 500 InternalServerError status
//
// The HTTP 500 Internal Server Error server error response status code indicates
// that the server encountered an unexpected condition that prevented it from fulfilling
// the request. This error is a generic "catch-all" response to server issues, indicating
// that the server cannot find a more appropriate 5XX error to respond with.
//
// If you're a visitor seeing 500 errors on a web page, these issues require investigation
// by server owners or administrators. There are many possible causes of 500 errors,
// including: improper server configuration, out-of-memory (OOM) issues, unhandled exceptions,
// improper file permissions, or other complex factors. Server administrators may proactively
// log occurrences of server error responses, like the 500 status code, with details about the
// initiating requests to improve the stability of a service in the future.
func internalErrorResponse(w http.ResponseWriter, message any) {
	errorResponse(w, http.StatusInternalServerError, message)
}
