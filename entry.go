package els

// Level constants represent error severity levels accepted by the ELS API.
const (
	LevelCritical = "critical"
	LevelError    = "error"
	LevelWarning  = "warning"
	LevelInfo     = "info"
	LevelDebug    = "debug"
)

// Source constants represent the origin of the error.
const (
	SourceClient = "client"
	SourceServer = "server"
)

// ErrorEntry represents a single error log entry to be sent to the ELS API.
// Fields marked as "auto" are filled automatically by the SDK if not provided.
type ErrorEntry struct {
	// Message is the error text (required).
	Message string `json:"message"`

	// URL is the request URL or page URL where the error occurred (required).
	URL string `json:"url"`

	// Timestamp in RFC3339 format (auto-filled if empty).
	Timestamp string `json:"timestamp"`

	// Stack is the stack trace. Auto-captured for CaptureError calls.
	Stack string `json:"stack,omitempty"`

	// ComponentStack is a framework-specific component trace (e.g. React).
	ComponentStack string `json:"componentStack,omitempty"`

	// Level is the severity: critical, error, warning, info, debug.
	// Default: "error".
	Level string `json:"level"`

	// Source indicates origin: "client" or "server". Default: "server".
	Source string `json:"source"`

	// UserAgent of the client that triggered the error.
	UserAgent string `json:"userAgent,omitempty"`

	// Language is the Accept-Language or locale of the client.
	Language string `json:"language,omitempty"`

	// ScreenSize in "WxH" format (e.g., "1920x1080").
	ScreenSize string `json:"screenSize,omitempty"`

	// ViewportSize in "WxH" format.
	ViewportSize string `json:"viewportSize,omitempty"`

	// Referrer is the HTTP Referer header value.
	Referrer string `json:"referrer,omitempty"`

	// ServiceName identifies the microservice (from config if not set).
	ServiceName string `json:"serviceName,omitempty"`

	// DeploymentEnv is the deployment environment (from config if not set).
	// Will be normalized server-side (e.g., "dev" → "DEV").
	DeploymentEnv string `json:"deploymentEnv,omitempty"`

	// AppSlug identifies the application (from config if not set).
	AppSlug string `json:"appSlug,omitempty"`

	// SessionID groups errors from the same logical session (auto-generated).
	SessionID string `json:"sessionId,omitempty"`

	// AppVersion is the version of the application sending this log entry.
	// Opaque string of any format (set from Config.AppVersion if empty).
	AppVersion string `json:"appVersion,omitempty"`

	// Meta holds arbitrary key-value metadata.
	Meta map[string]any `json:"meta,omitempty"`
}

// BatchResult is the response from the batch ingest endpoint.
type BatchResult struct {
	Created int    `json:"created"`
	Error   string `json:"error,omitempty"`
}

// batchRequest is the internal request payload for the batch endpoint.
type batchRequest struct {
	Errors []ErrorEntry `json:"errors"`
}
