package wrap

import (
	"context"
	"errors"
)

// WrapError wraps an error with the current LogCtx from the context
func Error(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// If already wrapped, just update logCtx
	var e *errorWithLogCtx
	if errors.As(err, &e) {
		if x, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
			e.logCtx = x
		}
		e.err = err
		return e
	}

	c := LogCtx{}
	if x, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
		c = x
	}
	return &errorWithLogCtx{
		err:    err,
		logCtx: c,
	}
}
