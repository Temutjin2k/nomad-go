package rabbit

import "time"

// isRecoverableError returns true if the provided error must be requeued
func isRecoverableError(err error) bool {
	return true
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
