package els

// Level is a typed error severity accepted by the ELS API. Use the
// Level* constants instead of raw strings.
type Level string

// Level constants represent error severity levels accepted by the ELS API.
const (
	LevelCritical Level = "critical"
	LevelError    Level = "error"
	LevelWarning  Level = "warning"
	LevelInfo     Level = "info"
	LevelDebug    Level = "debug"
)

// String returns the wire (JSON) value of the level.
func (l Level) String() string { return string(l) }

// Valid reports whether l is one of the known ELS levels.
func (l Level) Valid() bool {
	switch l {
	case LevelDebug, LevelInfo, LevelWarning, LevelError, LevelCritical:
		return true
	default:
		return false
	}
}

// Priority returns the numeric ordering of the level
// (debug < info < warning < error < critical). Unknown levels return -1.
func (l Level) Priority() int {
	switch l {
	case LevelDebug:
		return 0
	case LevelInfo:
		return 1
	case LevelWarning:
		return 2
	case LevelError:
		return 3
	case LevelCritical:
		return 4
	default:
		return -1
	}
}

// Source constants represent the origin of the error.
const (
	SourceClient = "client"
	SourceServer = "server"
)

// ErrorEntry represents a single error log entry sent to the ELS API.
//
// Message is the only field you must set. Everything else is optional: the SDK
// fills sensible defaults (TraceId/Timestamp/SessionID, plus AppSlug,
// ServiceName, DeploymentEnv and AppVersion from Config) and CaptureError adds
// a stack trace automatically. Most code never builds an ErrorEntry directly —
// use CaptureError / CaptureMessage with WithX options instead. Build one
// explicitly only for CaptureEntry / SendSyncEntry.
type ErrorEntry struct {
	// Message is the error text. This is the only required field.
	Message string `json:"message"`

	// URL is the request or page URL where the error occurred. Optional;
	// set it via WithURL or WithRequest, otherwise it is sent empty.
	URL string `json:"url"`

	// Timestamp in RFC3339 format (auto-filled if empty).
	Timestamp string `json:"timestamp"`

	// Stack is the stack trace. Auto-captured for CaptureError calls.
	Stack string `json:"stack,omitempty"`

	// ComponentStack is a framework-specific component trace (e.g. React).
	ComponentStack string `json:"componentStack,omitempty"`

	// Level is the severity: critical, error, warning, info, debug.
	// Default: "error".
	Level Level `json:"level"`

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
	Errors []*ErrorEntry `json:"errors"`
}
