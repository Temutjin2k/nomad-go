package rabbit

import (
	"errors"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
)

// isRecoverableError returns true if the provided error must be requeued
func isRecoverableError(err error) bool {
	return oneOf(err, types.ErrDatabaseFailed, types.ErrFailedToPublishRideStatus)
}

func oneOf(err error, targets ...error) bool {
	for _, t := range targets {
		if errors.Is(err, t) {
			return true
		}
	}
	return false
}

func retry(n int, sleep time.Duration, fn func() error) error {
	var err error
	for range n {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(sleep)
	}
	return err
}
