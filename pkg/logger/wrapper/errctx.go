package wrap

import (
	"context"
	"errors"
)

// errorWithLogCtx is a custom error type that wraps an error and includes LogCtx
type errorWithLogCtx struct {
	err    error
	logCtx LogCtx
}

// errorWithLogCtx implements the error interface
func (e *errorWithLogCtx) Error() string {
	return e.err.Error()
}

// Unwrap allows unwrapping the original error
func (e *errorWithLogCtx) Unwrap() error {
	return e.err
}

// ErrorCtx extracts the LogCtx from an error if it is of type errorWithLogCtx
func ErrorCtx(ctx context.Context, err error) context.Context {
	var e *errorWithLogCtx
	if errors.As(err, &e) && e != nil {
		return context.WithValue(ctx, LogCtxKey, e.logCtx)
	}
	return ctx
}
