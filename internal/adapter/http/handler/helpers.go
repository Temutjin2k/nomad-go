package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	t "github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	authSvc "github.com/Temutjin2k/ride-hail-system/internal/service/auth"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
	"github.com/jackc/pgx/v5"
)

type envelope map[string]any

func writeJSON(w http.ResponseWriter, status int, data any, headers http.Header) error {
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

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// Decode the request body to the destination.
	if err := dec.Decode(dst); err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		// Add a new maxBytesError variable.
		var maxBytesError *http.MaxBytesError
		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		// If the JSON contains a field which cannot be mapped to the target destination
		// then Decode() will now return an error message in the format "json: unknown
		// field "<name>"". We check for this, extract the field name from the error,
		// and interpolate it into our custom error message. Note that there's an open
		// issue at https://github.com/golang/go/issues/29035 regarding turning this
		// into a distinct error type in the future.
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		// Use the errors.As() function to check whether the error has the type
		// *http.MaxBytesError. If it does, then it means the request body exceeded our
		// size limit of 1MB and we return a clear error message.
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)
		case errors.As(err, &invalidUnmarshalError):
			return fmt.Errorf("invalid unmarshal error: %w", err)
		default:
			return err
		}
	}

	// Call Decode() again, using a pointer to an empty anonymous struct as the
	// destination. If the request body only contained a single JSON value this will
	// return an io.EOF error. So if we get anything else, we know that there is
	// additional data in the request body and we return our own custom error message.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func GetCode(err error) int {
	switch {
	case oneOf(err,
		t.ErrInvalidLicenseFormat,
		t.ErrDriverAlreadyOffline,
		t.ErrDriverAlreadyOnline,
		t.ErrLicenseAlreadyExists,
	):
		return http.StatusBadRequest

	case oneOf(err,
		t.ErrUserNotFound,
		t.ErrSessionNotFound,
		t.ErrNoCoordinates,
		t.ErrRideNotFound,
		t.ErrDriverLocationNotFound,
		sql.ErrNoRows,
		pgx.ErrNoRows,
	):
		return http.StatusNotFound

	case oneOf(err,
		t.ErrDriverRegistered,
		t.ErrDriverMustBeAvailable,
		authSvc.ErrNotUniqueEmail,
		t.ErrDriverAlreadyOnRide,
		t.ErrRideDriverMismatch,
		t.ErrRideNotArrived,
		t.ErrDriverMustBeEnRoute,
		t.ErrRideNotInProgress,
		t.ErrRideCannotBeCancelled,
		t.ErrDriverMustBeBusy,
	):
		return http.StatusConflict

	case oneOf(err,
		authSvc.ErrInvalidCredentials,
		authSvc.ErrInvalidToken,
		authSvc.ErrExpToken,
	):
		return http.StatusUnauthorized

	case oneOf(err, authSvc.ErrCannotCreateAdmin):
		return http.StatusForbidden

	default:
		return http.StatusInternalServerError
	}
}

func oneOf(err error, targets ...error) bool {
	for _, t := range targets {
		if errors.Is(err, t) {
			return true
		}
	}
	return false
}

// The readString() helper returns a string value from the query string, or the provided
// default value if no matching key could be found.
func readString(qs url.Values, key string, defaultValue string) string {
	// Extract the value for a given key from the query string. If no key exists this
	// will return the empty string "".
	s := qs.Get(key)

	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}

	// Otherwise return the string.
	return s
}

// The readInt() helper reads a string value from the query string and converts it to an
// integer before returning. If no matching key could be found it returns the provided
// default value. If the value couldn't be converted to an integer, then we record an
// error message in the provided Validator instance.
func readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	// Extract the value from the query string.
	s := qs.Get(key)

	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}

	// Try to convert the value to an int. If this fails, add an error message to the
	// validator instance and return the default value.
	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	// Otherwise, return the converted integer value.
	return i
}
