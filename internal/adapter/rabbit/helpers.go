package rabbit

// isRecoverableError returns true if the provided error must be requeued
func isRecoverableError(err error) bool {
	return true
}
