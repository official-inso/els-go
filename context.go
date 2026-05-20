package els

import (
	"context"
	"net/http"
	"time"
)

// ctxKey is a private type for context keys to avoid collisions.
type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyTraceID
)

// ContextWithRequestID returns a context carrying a request ID that the
// Capture*Ctx methods attach to entries as Meta["requestId"].
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, id)
}

// ContextWithTraceID returns a context carrying a trace ID that the
// Capture*Ctx methods attach to entries as Meta["traceId"].
func ContextWithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyTraceID, id)
}

func requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKeyRequestID).(string)
	return v
}

func traceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKeyTraceID).(string)
	return v
}

// ctxOption builds a CaptureOption that copies request/trace IDs from ctx into
// the entry's Meta.
func ctxOption(ctx context.Context) CaptureOption {
	return func(e *ErrorEntry) {
		rid := requestIDFromContext(ctx)
		tid := traceIDFromContext(ctx)
		if rid == "" && tid == "" {
			return
		}
		if e.Meta == nil {
			e.Meta = make(map[string]any)
		}
		if rid != "" {
			e.Meta["requestId"] = rid
		}
		if tid != "" {
			e.Meta["traceId"] = tid
		}
	}
}

// CaptureErrorCtx is like CaptureError but also copies request/trace IDs from
// ctx (set via ContextWithRequestID / ContextWithTraceID) into the entry.
func (c *Client) CaptureErrorCtx(ctx context.Context, err error, opts ...CaptureOption) {
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
	ctxOption(ctx)(entry)
	for _, opt := range opts {
		opt(entry)
	}
	c.enqueue(entry)
}

// CaptureMessageCtx is like CaptureMessage but also copies request/trace IDs
// from ctx into the entry.
func (c *Client) CaptureMessageCtx(ctx context.Context, msg string, level Level, opts ...CaptureOption) {
	all := make([]CaptureOption, 0, len(opts)+1)
	all = append(all, ctxOption(ctx))
	all = append(all, opts...)
	c.CaptureMessage(msg, level, all...)
}

// UserContext holds information about the current user.
// When set via SetUser, it is automatically attached to all captured entries via Meta.
type UserContext struct {
	// ID is the user's unique identifier.
	ID string
	// Email is the user's email address (optional).
	Email string
	// Name is the user's display name (optional).
	Name string
	// Extra holds additional user-specific key-value pairs.
	// These are added to Meta as "user.<key>".
	Extra map[string]string
}

// SetUser sets the user context for this client. All subsequent captures
// will include user information in the Meta field. Pass nil to clear.
//
//	client.SetUser(&els.UserContext{
//	    ID:    "usr_123",
//	    Email: "john@example.com",
//	    Name:  "John Doe",
//	    Extra: map[string]string{"tenant": "acme-corp"},
//	})
func (c *Client) SetUser(user *UserContext) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.user = user
}

// User returns the currently set user context, or nil.
func (c *Client) User() *UserContext {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.user
}

// WithRequest extracts URL, UserAgent, Referrer, Language, and method from
// an *http.Request and attaches them to the error entry. This is the
// recommended way to capture HTTP context in your handlers.
//
//	client.CaptureError(err, els.WithRequest(r))
func WithRequest(r *http.Request) CaptureOption {
	return func(e *ErrorEntry) {
		if r == nil {
			return
		}
		if e.URL == "" {
			e.URL = r.Method + " " + r.URL.String()
		}
		if e.UserAgent == "" {
			e.UserAgent = r.UserAgent()
		}
		if e.Referrer == "" {
			e.Referrer = r.Referer()
		}
		if e.Language == "" {
			if lang := r.Header.Get("Accept-Language"); lang != "" {
				// Take the first language preference
				if idx := len(lang); idx > 20 {
					lang = lang[:20]
				}
				e.Language = lang
			}
		}
		// Enrich meta with request details
		if e.Meta == nil {
			e.Meta = make(map[string]any)
		}
		e.Meta["http.method"] = r.Method
		e.Meta["http.host"] = r.Host
		e.Meta["http.remoteAddr"] = r.RemoteAddr
		if rid := r.Header.Get("X-Request-Id"); rid != "" {
			e.Meta["http.requestId"] = rid
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			e.Meta["http.forwardedFor"] = xff
		}
	}
}

// WithCause walks the error chain via Unwrap() and stores the cause messages
// in Meta as "error.causes". This preserves the full error context for debugging.
//
//	err := fmt.Errorf("handler failed: %w", dbErr)
//	client.CaptureError(err, els.WithCause(err))
func WithCause(err error) CaptureOption {
	return func(e *ErrorEntry) {
		if err == nil {
			return
		}
		type unwrapper interface {
			Unwrap() error
		}

		var causes []string
		current := err
		for {
			u, ok := current.(unwrapper)
			if !ok {
				break
			}
			current = u.Unwrap()
			if current == nil {
				break
			}
			causes = append(causes, current.Error())
		}

		if len(causes) > 0 {
			if e.Meta == nil {
				e.Meta = make(map[string]any)
			}
			e.Meta["error.causes"] = causes
		}
	}
}
