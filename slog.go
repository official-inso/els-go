package els

import (
	"context"
	"log/slog"
	"runtime"
	"strconv"
)

// SlogHandler returns a slog.Handler that sends log records to ELS.
// Use this to integrate ELS with Go's standard structured logging.
//
//	logger := slog.New(els.SlogHandler(client, nil))
//	slog.SetDefault(logger)
//
//	slog.Error("database timeout", "db", "postgres", "latency_ms", 5000)
//	// → captured as ELS error with meta: {"db": "postgres", "latency_ms": 5000}
//
// The handler respects MinLevel from ELS config. Pass a custom slog.HandlerOptions
// to further configure behavior (e.g., add source location).
func SlogHandler(client *Client, opts *SlogHandlerOptions) slog.Handler {
	if opts == nil {
		opts = &SlogHandlerOptions{}
	}
	return &elsSlogHandler{
		client: client,
		opts:   *opts,
	}
}

// SlogHandlerOptions configures the slog handler behavior.
type SlogHandlerOptions struct {
	// AddSource adds source file and line number to error entries.
	// Default: true.
	AddSource bool

	// URL is the default URL attached to entries when none is provided via attributes.
	// Default: "slog".
	URL string
}

type elsSlogHandler struct {
	client *Client
	opts   SlogHandlerOptions
	attrs  []slog.Attr
	groups []string
}

func (h *elsSlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	elsLevel := slogLevelToELS(level)
	return h.client.shouldCapture(elsLevel)
}

func (h *elsSlogHandler) Handle(_ context.Context, record slog.Record) error {
	level := slogLevelToELS(record.Level)

	entry := &ErrorEntry{
		Message:   record.Message,
		Level:     level,
		Source:    h.client.config.DefaultSource,
		Timestamp: record.Time.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		SessionID: h.client.SessionID(),
		URL:       h.opts.URL,
	}

	if entry.URL == "" {
		entry.URL = "slog"
	}

	// Add source location
	if h.opts.AddSource {
		if record.PC != 0 {
			fs := runtime.CallersFrames([]uintptr{record.PC})
			f, _ := fs.Next()
			if f.File != "" {
				entry.Stack = f.Function + "\n\t" + f.File + ":" + strconv.Itoa(f.Line) + "\n"
			}
		}
	}

	// Collect attributes into Meta
	meta := make(map[string]any)

	// Pre-existing attrs from WithAttrs
	for _, a := range h.attrs {
		prefix := groupPrefix(h.groups)
		meta[prefix+a.Key] = a.Value.Any()
	}

	// Record-level attrs
	record.Attrs(func(a slog.Attr) bool {
		prefix := groupPrefix(h.groups)
		meta[prefix+a.Key] = a.Value.Any()
		return true
	})

	if len(meta) > 0 {
		entry.Meta = meta
	}

	h.client.enqueue(entry)
	return nil
}

func (h *elsSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	return &elsSlogHandler{
		client: h.client,
		opts:   h.opts,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

func (h *elsSlogHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	return &elsSlogHandler{
		client: h.client,
		opts:   h.opts,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

func slogLevelToELS(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return LevelError
	case level >= slog.LevelWarn:
		return LevelWarning
	case level >= slog.LevelInfo:
		return LevelInfo
	default:
		return LevelDebug
	}
}

func groupPrefix(groups []string) string {
	if len(groups) == 0 {
		return ""
	}
	prefix := ""
	for _, g := range groups {
		prefix += g + "."
	}
	return prefix
}
