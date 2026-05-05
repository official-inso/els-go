package els

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Config holds the SDK configuration.
type Config struct {
	// Endpoint is the base URL of the ELS API (required).
	// Example: "https://api.example.com/els"
	Endpoint string

	// APIKey is the authentication key for the ELS API (required).
	APIKey string

	// AppSlug identifies the application sending errors (recommended).
	AppSlug string

	// DeploymentEnv is the deployment environment (e.g., "DEV", "PRODUCTION").
	DeploymentEnv string

	// ServiceName identifies the specific microservice.
	ServiceName string

	// BatchSize is the maximum number of entries per batch request.
	// Default: 50.
	BatchSize int

	// BatchInterval is the maximum time to wait before flushing a partial batch.
	// Default: 5 seconds.
	BatchInterval time.Duration

	// BufferSize is the capacity of the in-memory entry queue.
	// Default: 1000.
	BufferSize int

	// MaxRetries is the number of retry attempts for failed requests.
	// Default: 3.
	MaxRetries int

	// RetryBaseDelay is the initial delay between retries (doubles each attempt).
	// Default: 1 second.
	RetryBaseDelay time.Duration

	// Timeout is the HTTP request timeout.
	// Default: 10 seconds.
	Timeout time.Duration

	// BufferDir is the directory for disk-based buffering when the server is unreachable.
	// Default: os.TempDir().
	BufferDir string

	// BeforeSend is called before each entry is enqueued.
	// Return nil to drop the entry. Modify the entry in-place to mutate it.
	BeforeSend func(*ErrorEntry) *ErrorEntry

	// OnError is called when the SDK encounters an internal error
	// (e.g., failed to send after all retries, disk write error).
	OnError func(err error)

	// DefaultLevel is the default severity level for captured entries.
	// Default: "error".
	DefaultLevel string

	// DefaultSource is the default source for captured entries.
	// Default: "server".
	DefaultSource string

	// Debug enables verbose internal logging to stderr.
	Debug bool
}

func (c *Config) applyDefaults() {
	if c.BatchSize <= 0 {
		c.BatchSize = 50
	}
	if c.BatchInterval <= 0 {
		c.BatchInterval = 5 * time.Second
	}
	if c.BufferSize <= 0 {
		c.BufferSize = 1000
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}
	if c.RetryBaseDelay <= 0 {
		c.RetryBaseDelay = time.Second
	}
	if c.Timeout <= 0 {
		c.Timeout = 10 * time.Second
	}
	if c.DefaultLevel == "" {
		c.DefaultLevel = LevelError
	}
	if c.DefaultSource == "" {
		c.DefaultSource = SourceServer
	}
}

// Client is the main ELS SDK instance. It is safe for concurrent use.
type Client struct {
	config    Config
	queue     chan *ErrorEntry
	transport *httpTransport
	wg        sync.WaitGroup
	done      chan struct{}
	mu        sync.RWMutex
	closed    bool
	sessionID string
}

// New creates and starts a new ELS Client. The background worker begins
// immediately and will batch and send entries as they are captured.
// Call Close() to gracefully shut down.
func New(config Config) (*Client, error) {
	if config.Endpoint == "" {
		return nil, errors.New("els: Endpoint is required")
	}
	if config.APIKey == "" {
		return nil, errors.New("els: APIKey is required")
	}

	config.applyDefaults()

	c := &Client{
		config:    config,
		queue:     make(chan *ErrorEntry, config.BufferSize),
		done:      make(chan struct{}),
		sessionID: generateSessionID(),
		transport: newHTTPTransport(config),
	}

	c.wg.Add(1)
	go c.worker()

	return c, nil
}

// CaptureError captures an error with an automatic stack trace and optional options.
// Returns immediately; the entry is sent asynchronously.
func (c *Client) CaptureError(err error, opts ...CaptureOption) {
	if err == nil {
		return
	}

	entry := &ErrorEntry{
		Message:   err.Error(),
		Stack:     captureStack(3),
		Level:     c.config.DefaultLevel,
		Source:    c.config.DefaultSource,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		SessionID: c.SessionID(),
	}

	for _, opt := range opts {
		opt(entry)
	}

	c.enqueue(entry)
}

// CaptureMessage captures a text message at the given level.
func (c *Client) CaptureMessage(msg string, level string, opts ...CaptureOption) {
	entry := &ErrorEntry{
		Message:   msg,
		Level:     level,
		Source:    c.config.DefaultSource,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		SessionID: c.SessionID(),
	}

	for _, opt := range opts {
		opt(entry)
	}

	c.enqueue(entry)
}

// CaptureEntry sends a pre-built ErrorEntry. Missing fields are filled with defaults.
func (c *Client) CaptureEntry(entry ErrorEntry, opts ...CaptureOption) {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if entry.Level == "" {
		entry.Level = c.config.DefaultLevel
	}
	if entry.Source == "" {
		entry.Source = c.config.DefaultSource
	}
	if entry.SessionID == "" {
		entry.SessionID = c.SessionID()
	}

	for _, opt := range opts {
		opt(&entry)
	}

	c.enqueue(&entry)
}

// Flush blocks until the current queue is drained or 10 seconds elapse.
func (c *Client) Flush() {
	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			return
		default:
			if len(c.queue) == 0 {
				time.Sleep(100 * time.Millisecond)
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// Close performs graceful shutdown: signals the worker to stop, drains the
// queue, flushes remaining entries, and persists any unsent entries to disk.
func (c *Client) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()

	close(c.done)
	c.wg.Wait()
}

// SessionID returns the process-level session identifier.
func (c *Client) SessionID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionID
}

// SetSessionID overrides the auto-generated session ID.
func (c *Client) SetSessionID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = id
}

func (c *Client) enqueue(entry *ErrorEntry) {
	c.mu.RLock()
	closed := c.closed
	c.mu.RUnlock()

	if closed {
		return
	}

	// Enrich with config defaults
	if entry.AppSlug == "" {
		entry.AppSlug = c.config.AppSlug
	}
	if entry.DeploymentEnv == "" {
		entry.DeploymentEnv = c.config.DeploymentEnv
	}
	if entry.ServiceName == "" {
		entry.ServiceName = c.config.ServiceName
	}

	// BeforeSend hook
	if c.config.BeforeSend != nil {
		entry = c.config.BeforeSend(entry)
		if entry == nil {
			return
		}
	}

	select {
	case c.queue <- entry:
	default:
		// Queue full: drop oldest entry and push new one
		select {
		case <-c.queue:
		default:
		}
		select {
		case c.queue <- entry:
		default:
		}
	}
}

func generateSessionID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("els-%d", time.Now().UnixNano())
	}
	return "els-" + hex.EncodeToString(buf)
}

func captureStack(skip int) string {
	const maxFrames = 32
	pcs := make([]uintptr, maxFrames)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return ""
	}

	frames := runtime.CallersFrames(pcs[:n])
	var sb strings.Builder
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&sb, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	return sb.String()
}
