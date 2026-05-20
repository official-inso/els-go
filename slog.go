package els

import (
	"context"
	"log/slog"
	"runtime"
	"strconv"
)

// SlogHandler returns a slog.Handler that forwards log records to ELS, so an
// existing log/slog setup reports to ELS with no call-site changes. Pass nil
// for sensible defaults (AddSource and CaptureStackOnError enabled).
//
//	logger := slog.New(els.SlogHandler(client, nil))
//	slog.SetDefault(logger)
//
//	slog.Info("cache warm", "keys", 1280)
//	// A record carrying an error attribute (key "err"/"error") is captured WITH
//	// a full stack trace and the error text in Meta — just like CaptureError:
//	slog.Error("db query failed", "err", dbErr, "table", "users")
//
// Record attributes become entry Meta. The handler respects Config.MinLevel,
// and ServiceName/AppSlug/etc. are filled from Config automatically. Configure
// behavior via SlogHandlerOptions.
func SlogHandler(client *Client, opts *SlogHandlerOptions) slog.Handler {
	if opts == nil {
		// Sensible defaults when no options are provided.
		opts = &SlogHandlerOptions{
			AddSource:           true,
			CaptureStackOnError: true,
		}
	}
	if len(opts.ErrorKeys) == 0 {
		opts.ErrorKeys = []string{"err", "error"}
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

	// ErrorKeys lists the attribute keys whose value, when it is an error,
	// is treated as the cause of the log record. The handler then captures a
	// full stack trace (like CaptureError) and stores the error text in Meta.
	// Default: {"err", "error"}.
	ErrorKeys []string

	// CaptureStackOnError, when true, attaches a full stack trace to records
	// that carry an error attribute or whose level is >= error.
	// Default: true.
	CaptureStackOnError bool
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

	// Collect attributes into Meta and detect an error-valued attribute.
	meta := make(map[string]any)
	var capturedErr error

	collect := func(a slog.Attr) {
		prefix := groupPrefix(h.groups)
		v := a.Value.Any()
		meta[prefix+a.Key] = v
		if capturedErr == nil {
			if err, ok := v.(error); ok && h.isErrorKey(a.Key) {
				capturedErr = err
			}
		}
	}

	// Pre-existing attrs from WithAttrs, then record-level attrs.
	for _, a := range h.attrs {
		collect(a)
	}
	record.Attrs(func(a slog.Attr) bool {
		collect(a)
		return true
	})

	// Route error-carrying records like CaptureError: full stack + error text.
	isError := capturedErr != nil || level.Priority() >= LevelError.Priority()
	if capturedErr != nil {
		meta["error"] = capturedErr.Error()
	}

	switch {
	case h.opts.CaptureStackOnError && isError:
		// Full multi-frame stack (skips runtime.Callers + this frame).
		entry.Stack = captureStack(3)
	case h.opts.AddSource && record.PC != 0:
		// Single source frame for non-error records.
		fs := runtime.CallersFrames([]uintptr{record.PC})
		f, _ := fs.Next()
		if f.File != "" {
			entry.Stack = f.Function + "\n\t" + f.File + ":" + strconv.Itoa(f.Line) + "\n"
		}
	}

	if len(meta) > 0 {
		entry.Meta = meta
	}

	h.client.enqueue(entry)
	return nil
}

// isErrorKey reports whether key is configured as an error-carrying attribute.
func (h *elsSlogHandler) isErrorKey(key string) bool {
	for _, k := range h.opts.ErrorKeys {
		if k == key {
			return true
		}
	}
	return false
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

func slogLevelToELS(level slog.Level) Level {
	return LevelFromSlog(level)
}

// LevelFromSlog maps a slog.Level to the corresponding ELS Level.
func LevelFromSlog(level slog.Level) Level {
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

// ToSlog maps an ELS Level back to the nearest slog.Level.
func (l Level) ToSlog() slog.Level {
	switch l {
	case LevelCritical, LevelError:
		return slog.LevelError
	case LevelWarning:
		return slog.LevelWarn
	case LevelInfo:
		return slog.LevelInfo
	default:
		return slog.LevelDebug
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
