package els

// Logger-style shorthands for CaptureMessage at a fixed level. They are
// non-blocking (enqueue + return), like CaptureMessage.
//
// Note: these capture a *message* at the given level. To capture a Go error
// value together with an automatic stack trace, use CaptureError instead.

// Debug captures a debug-level message.
func (c *Client) Debug(msg string, opts ...CaptureOption) {
	c.CaptureMessage(msg, LevelDebug, opts...)
}

// Info captures an info-level message.
func (c *Client) Info(msg string, opts ...CaptureOption) {
	c.CaptureMessage(msg, LevelInfo, opts...)
}

// Warning captures a warning-level message.
func (c *Client) Warning(msg string, opts ...CaptureOption) {
	c.CaptureMessage(msg, LevelWarning, opts...)
}

// Error captures an error-level message. For a Go error value with a stack
// trace, prefer CaptureError.
func (c *Client) Error(msg string, opts ...CaptureOption) {
	c.CaptureMessage(msg, LevelError, opts...)
}

// Critical captures a critical-level message.
func (c *Client) Critical(msg string, opts ...CaptureOption) {
	c.CaptureMessage(msg, LevelCritical, opts...)
}
