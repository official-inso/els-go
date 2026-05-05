package els

import "fmt"

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

// IsRetryableErr returns true if the error is a retryable SendError.
func IsRetryableErr(err error) bool {
	var se *SendError
	if As(err, &se) {
		return se.IsRetryable
	}
	return false
}

// As is a convenience re-export of errors.As for use with SendError.
func As(err error, target interface{}) bool {
	type asInterface interface {
		As(interface{}) bool
	}
	// Walk the error chain
	for err != nil {
		if se, ok := target.(**SendError); ok {
			if v, ok2 := err.(*SendError); ok2 {
				*se = v
				return true
			}
		}
		if x, ok := err.(asInterface); ok {
			if x.As(target) {
				return true
			}
		}
		type unwrapper interface {
			Unwrap() error
		}
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
