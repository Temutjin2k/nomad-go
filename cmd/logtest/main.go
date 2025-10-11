package main

import (
	"context"
	"errors"

	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
)

func main() {
	logger.Example()
}

func SomeLogic(ctx context.Context) error {
	ctx = wrap.WithLogCtx(ctx, wrap.LogCtx{
		Action:    "test",
		UserID:    "123",
		RequestID: "request_123",
	})

	someError := errors.New("some error")

	return wrap.Error(ctx, someError)
}
