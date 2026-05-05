package els

// CaptureOption is a functional option applied to an ErrorEntry before it is enqueued.
type CaptureOption func(*ErrorEntry)

// WithLevel sets the severity level of the error entry.
func WithLevel(level string) CaptureOption {
	return func(e *ErrorEntry) { e.Level = level }
}

// WithSource sets the error source (client or server).
func WithSource(source string) CaptureOption {
	return func(e *ErrorEntry) { e.Source = source }
}

// WithURL sets the URL where the error occurred.
func WithURL(url string) CaptureOption {
	return func(e *ErrorEntry) { e.URL = url }
}

// WithStack overrides the auto-captured stack trace.
func WithStack(stack string) CaptureOption {
	return func(e *ErrorEntry) { e.Stack = stack }
}

// WithMeta sets arbitrary metadata on the entry.
func WithMeta(meta map[string]any) CaptureOption {
	return func(e *ErrorEntry) { e.Meta = meta }
}

// WithUserAgent sets the user agent string.
func WithUserAgent(ua string) CaptureOption {
	return func(e *ErrorEntry) { e.UserAgent = ua }
}

// WithLanguage sets the client language/locale.
func WithLanguage(lang string) CaptureOption {
	return func(e *ErrorEntry) { e.Language = lang }
}

// WithSessionID overrides the auto-generated session ID for this entry.
func WithSessionID(id string) CaptureOption {
	return func(e *ErrorEntry) { e.SessionID = id }
}

// WithServiceName overrides the configured service name for this entry.
func WithServiceName(name string) CaptureOption {
	return func(e *ErrorEntry) { e.ServiceName = name }
}

// WithReferrer sets the HTTP referrer.
func WithReferrer(ref string) CaptureOption {
	return func(e *ErrorEntry) { e.Referrer = ref }
}

// WithComponentStack sets a framework-level component trace.
func WithComponentStack(cs string) CaptureOption {
	return func(e *ErrorEntry) { e.ComponentStack = cs }
}
