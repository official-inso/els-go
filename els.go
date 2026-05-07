package els

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	mathrand "math/rand"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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

	// AppVersion is the version of the application sending logs. Any string
	// up to 128 chars is accepted: semver ("1.2.3"), CalVer ("2026.05.07"),
	// date-compact ("20260507120000"), git SHA, prefixed ("v1.2.3"), opaque.
	// ELS analytics auto-detects the format and sorts versions in timeline.
	// Recommended: pass `os.Getenv("BUILD_VERSION")` set by your build pipeline.
	AppVersion string

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

	// FlushTimeout is the maximum time Flush() will wait for the queue to drain.
	// Default: 10 seconds.
	FlushTimeout time.Duration

	// BufferDir is the directory for disk-based buffering when the server is unreachable.
	// Default: os.TempDir().
	BufferDir string

	// MaxBufferFileSize is the maximum size of the disk buffer file in bytes.
	// When exceeded, oldest entries are discarded. Default: 100 MB.
	MaxBufferFileSize int64

	// MinLevel is the minimum severity level to capture. Entries below this level
	// are silently dropped. Order: debug < info < warning < error < critical.
	// Default: "" (capture all levels).
	MinLevel string

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

	// HTTPClient allows providing a custom *http.Client for all ELS requests.
	// Use this to configure custom TLS, proxies, or request middleware.
	// Default: &http.Client{Timeout: Config.Timeout}.
	HTTPClient *http.Client

	// SampleRate controls what fraction of entries are actually sent (0.0 to 1.0).
	// 1.0 means send everything (default). 0.5 means send ~50% of entries.
	// Critical-level entries are never sampled (always sent).
	// Default: 1.0.
	SampleRate float64

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
	if c.FlushTimeout <= 0 {
		c.FlushTimeout = 10 * time.Second
	}
	if c.MaxBufferFileSize <= 0 {
		c.MaxBufferFileSize = 100 * 1024 * 1024 // 100 MB
	}
	if c.DefaultLevel == "" {
		c.DefaultLevel = LevelError
	}
	if c.DefaultSource == "" {
		c.DefaultSource = SourceServer
	}
	if c.SampleRate <= 0 || c.SampleRate > 1.0 {
		c.SampleRate = 1.0
	}
}

// Client is the main ELS SDK instance. It is safe for concurrent use.
type Client struct {
	config    Config
	queue     chan *ErrorEntry
	transport *httpTransport
	wg        sync.WaitGroup
	done      chan struct{}
	closed    atomic.Bool
	mu        sync.RWMutex
	sessionID string
	user      *UserContext
	stats     Stats
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

// SendSync sends a single error synchronously, waiting for server confirmation.
// Use this for critical errors where delivery must be guaranteed (e.g., payment failures).
// Unlike CaptureError, this blocks until the entry is delivered or the context expires.
func (c *Client) SendSync(ctx context.Context, err error, opts ...CaptureOption) error {
	if err == nil {
		return nil
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

	c.enrichDefaults(entry)

	if c.config.BeforeSend != nil {
		entry = c.config.BeforeSend(entry)
		if entry == nil {
			return nil
		}
	}

	if !c.shouldCapture(entry.Level) {
		return nil
	}

	return c.transport.sendSingle(ctx, *entry)
}

// SendSyncEntry sends a pre-built entry synchronously.
func (c *Client) SendSyncEntry(ctx context.Context, entry ErrorEntry, opts ...CaptureOption) error {
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

	c.enrichDefaults(&entry)

	if c.config.BeforeSend != nil {
		result := c.config.BeforeSend(&entry)
		if result == nil {
			return nil
		}
		entry = *result
	}

	if !c.shouldCapture(entry.Level) {
		return nil
	}

	return c.transport.sendSingle(ctx, entry)
}

// Health checks connectivity to the ELS server. Returns nil if the server is reachable.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.Endpoint+"/health", nil)
	if err != nil {
		return fmt.Errorf("els: create health request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.transport.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("els: health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("els: health check returned status %d", resp.StatusCode)
	}
	return nil
}

// Flush blocks until the current queue is drained or FlushTimeout elapses.
func (c *Client) Flush() {
	c.FlushWithTimeout(c.config.FlushTimeout)
}

// FlushWithTimeout blocks until the queue is drained or the given timeout elapses.
func (c *Client) FlushWithTimeout(timeout time.Duration) {
	deadline := time.After(timeout)
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
// Implements io.Closer.
func (c *Client) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil // already closed
	}

	close(c.done)
	c.wg.Wait()
	return nil
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

// --- Internal ---

func (c *Client) enqueue(entry *ErrorEntry) {
	if c.closed.Load() {
		return
	}

	if !c.shouldCapture(entry.Level) {
		return
	}

	// Sampling: critical entries always pass, others sampled by SampleRate
	if c.config.SampleRate < 1.0 && entry.Level != LevelCritical {
		if !c.shouldSample() {
			atomic.AddInt64(&c.stats.Sampled, 1)
			return
		}
	}

	c.enrichDefaults(entry)
	c.enrichUserContext(entry)

	// BeforeSend hook
	if c.config.BeforeSend != nil {
		entry = c.config.BeforeSend(entry)
		if entry == nil {
			return
		}
	}

	// Non-blocking send to queue. Uses done channel as secondary guard
	// to prevent sending after Close() has been called.
	select {
	case c.queue <- entry:
		atomic.AddInt64(&c.stats.Enqueued, 1)
	case <-c.done:
		return
	default:
		// Queue full: drop oldest entry and push new one
		atomic.AddInt64(&c.stats.Dropped, 1)
		select {
		case <-c.queue:
		default:
		}
		select {
		case c.queue <- entry:
			atomic.AddInt64(&c.stats.Enqueued, 1)
		case <-c.done:
		default:
		}
	}
}

func (c *Client) enrichDefaults(entry *ErrorEntry) {
	if entry.AppSlug == "" {
		entry.AppSlug = c.config.AppSlug
	}
	if entry.DeploymentEnv == "" {
		entry.DeploymentEnv = c.config.DeploymentEnv
	}
	if entry.ServiceName == "" {
		entry.ServiceName = c.config.ServiceName
	}
	if entry.AppVersion == "" {
		entry.AppVersion = c.config.AppVersion
	}
}

// levelPriority returns numeric priority for level comparison.
var levelPriority = map[string]int{
	LevelDebug:    0,
	LevelInfo:     1,
	LevelWarning:  2,
	LevelError:    3,
	LevelCritical: 4,
}

// shouldCapture returns true if the entry's level meets the MinLevel threshold.
func (c *Client) shouldCapture(level string) bool {
	if c.config.MinLevel == "" {
		return true
	}
	minP, ok := levelPriority[c.config.MinLevel]
	if !ok {
		return true
	}
	entryP, ok := levelPriority[level]
	if !ok {
		return true
	}
	return entryP >= minP
}

func generateSessionID() string {
	buf := make([]byte, 16)
	if _, err := cryptorand.Read(buf); err != nil {
		return fmt.Sprintf("els-%d", time.Now().UnixNano())
	}
	return "els-" + hex.EncodeToString(buf)
}

// shouldSample returns true based on the configured SampleRate probability.
func (c *Client) shouldSample() bool {
	return mathrand.Float64() < c.config.SampleRate
}

// enrichUserContext attaches user info to the entry's Meta if a user is set.
func (c *Client) enrichUserContext(entry *ErrorEntry) {
	c.mu.RLock()
	user := c.user
	c.mu.RUnlock()

	if user == nil {
		return
	}

	if entry.Meta == nil {
		entry.Meta = make(map[string]any)
	}
	if user.ID != "" {
		entry.Meta["user.id"] = user.ID
	}
	if user.Email != "" {
		entry.Meta["user.email"] = user.Email
	}
	if user.Name != "" {
		entry.Meta["user.name"] = user.Name
	}
	for k, v := range user.Extra {
		entry.Meta["user."+k] = v
	}
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
