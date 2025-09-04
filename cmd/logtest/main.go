package main

import (
	"context"
	"errors"

	l "github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

func main() {
	lg := l.InitLogger("test", l.LevelDebug)

	ctx := context.Background()

	if err := SomeLogic(ctx); err != nil {
		lg.Error(l.ErrorCtx(ctx, err), "error occured", err)
	}
}

func SomeLogic(ctx context.Context) error {
	ctx = l.WithLogCtx(ctx, l.LogCtx{
		Action:    "test",
		UserID:    "123",
		RequestID: "request_123",
	})

	someError := errors.New("some error")

	return l.WrapError(ctx, someError)
}
