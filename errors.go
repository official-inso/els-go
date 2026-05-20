package els

import (
	stderrors "errors"
	"fmt"
)

// SendError represents an error returned by the ELS API during send operations.
// It distinguishes between retryable (server/network) and permanent (client) errors.
type SendError struct {
	// StatusCode is the HTTP status code returned by the server (0 for network errors).
	StatusCode int

	// IsRetryable indicates whether the error is transient and worth retrying.
	// True for: 5xx, 429, network errors. False for: 4xx (except 429).
	IsRetryable bool

	// Err is the underlying error.
	Err error
}

func (e *SendError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("els: HTTP %d: %v", e.StatusCode, e.Err)
	}
	return fmt.Sprintf("els: %v", e.Err)
}

func (e *SendError) Unwrap() error {
	return e.Err
}

// newRetryableError creates a retryable SendError (server/network failure).
func newRetryableError(statusCode int, err error) *SendError {
	return &SendError{StatusCode: statusCode, IsRetryable: true, Err: err}
}

// newPermanentError creates a non-retryable SendError (client error).
func newPermanentError(statusCode int, err error) *SendError {
	return &SendError{StatusCode: statusCode, IsRetryable: false, Err: err}
}

// IsRetryableErr reports whether err (or any error it wraps) is a retryable
// *SendError — i.e. a transient failure (5xx, 429, or a network error) that is
// safe to retry. Returns false for permanent client errors (4xx except 429) and
// for non-ELS errors.
//
//	if err := client.SendSync(ctx, myErr); err != nil {
//	    if els.IsRetryableErr(err) {
//	        // transient — safe to retry
//	    }
//	}
func IsRetryableErr(err error) bool {
	var se *SendError
	if stderrors.As(err, &se) {
		return se.IsRetryable
	}
	return false
}

// As is a thin convenience wrapper around the standard library's errors.As,
// kept for backward compatibility. Prefer errors.As directly in new code.
func As(err error, target any) bool {
	return stderrors.As(err, target)
}
