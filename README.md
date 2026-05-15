# ELS Go SDK

High-performance Go SDK for [Error Logs Service](https://github.com/official-inso/els-go). Zero dependencies, async batching, disk buffering.

[Русская версия](README_RU.md)

## What you get

ELS provides a built-in admin dashboard out of the box. All events sent from your Go application appear there with full search, filtering, AI diagnosis, and version-aware regression detection.

| | |
|---|---|
| ![Logs list](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/01-error-logs-list.png) | ![Event detail](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/02-event-detail-info.png) |
| Virtual table with facet sidebar (app, env, **version**, source, level, browser, IP, category) | Full event metadata: timestamps, geo, env, **app version**, fingerprint, session |
| ![AI diagnosis](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/03-error-detail-ai.png) | ![Analytics](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/04-analytics-dashboard.png) |
| Stack trace + AI diagnosis (what broke, where, how to fix) | Dashboard with timeline, donuts, **version regressions** widget |
| ![API keys](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/05-api-keys.png) | ![Favorites](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/07-favorites.png) |
| Scoped API keys (write/read/read-any), live/test envs, rotation | Bookmarks for trace IDs that survive across sessions |

## Install

```bash
go get github.com/official-inso/els-go
```

## Quick Start

```go
package main

import (
    "errors"
    "log"

    els "github.com/official-inso/els-go"
)

func main() {
    // Option A: global client (recommended for most apps)
    els.Init(els.Config{
        Endpoint:      "https://api.example.com/els",
        APIKey:        "your-api-key",
        AppSlug:       "my-service",
        DeploymentEnv: "PRODUCTION",
        AppVersion:    os.Getenv("BUILD_VERSION"), // see "Versioning" below
    })
    defer els.Close()

    els.CaptureErrorGlobal(errors.New("something broke"), els.WithURL("/api/users"))

    // Option B: explicit client (for libraries or multiple instances)
    client, err := els.New(els.Config{...})
    if err != nil { log.Fatal(err) }
    defer client.Close()

    client.CaptureError(errors.New("db timeout"), els.WithURL("/api/data"))
}
```

## Core Concepts

### Async vs Sync

Most captures are **async** — they return immediately and the entry is sent in the background:

```go
client.CaptureError(err, els.WithURL("/api"))       // non-blocking
client.CaptureMessage("started", els.LevelInfo)     // non-blocking
```

For critical errors where you need **delivery confirmation**, use `SendSync`:

```go
err := client.SendSync(ctx, paymentErr,
    els.WithURL("/api/pay"),
    els.WithLevel(els.LevelCritical),
)
if err != nil {
    // Handle: error was NOT delivered
}
```

### Options Pattern

Every capture method accepts options to enrich the entry:

```go
client.CaptureError(err,
    els.WithURL("/api/orders"),
    els.WithLevel(els.LevelCritical),
    els.WithMeta(map[string]any{"orderId": "123"}),
    els.WithRequest(httpReq),        // auto-extracts URL, UA, Referrer, headers
    els.WithCause(err),              // preserves error chain in meta
    els.WithSessionID("req-abc"),
)
```

Full list: `WithURL`, `WithLevel`, `WithSource`, `WithStack`, `WithMeta`, `WithUserAgent`, `WithLanguage`, `WithReferrer`, `WithSessionID`, `WithServiceName`, `WithComponentStack`, `WithRequest`, `WithCause`.

---

## Features

### HTTP Middleware

Automatically captures panics in your HTTP handlers:

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/data", handler)

// Captures panic + returns 500
http.ListenAndServe(":8080", client.RecoverMiddleware(mux))

// Captures panic + re-raises (use with your own recovery layer)
http.ListenAndServe(":8080", client.Middleware(mux))
```

### User Context

Attach user info to all subsequent captures:

```go
client.SetUser(&els.UserContext{
    ID:    "usr_123",
    Email: "john@example.com",
    Extra: map[string]string{"tenant": "acme"},
})
// All entries now include user.id, user.email, user.tenant in Meta
```

### Slog Integration

Route Go's standard `log/slog` to ELS:

```go
logger := slog.New(els.SlogHandler(client, nil))
slog.SetDefault(logger)

slog.Error("db timeout", "host", "pg-1", "latency_ms", 5200)
// → captured as ELS error with meta: {"host": "pg-1", "latency_ms": 5200}
```

### Sampling

Control volume in high-traffic production:

```go
els.New(els.Config{
    SampleRate: 0.1, // send ~10% of non-critical entries
})
```

Critical-level entries always pass regardless of sample rate.

### Level Filtering

Drop low-severity entries entirely:

```go
els.New(els.Config{
    MinLevel: els.LevelWarning, // drops debug and info
})
```

Priority: `debug` < `info` < `warning` < `error` < `critical`

### Health Check

```go
if err := client.Health(ctx); err != nil {
    log.Printf("ELS unreachable: %v", err)
}
```

### Disk Buffering

When the server is unreachable after retries, entries are saved to `.els-buffer.jsonl`. On next startup, they're automatically re-sent. Capped at `MaxBufferFileSize` (default 100MB).

### Metrics

```go
stats := client.GetStats()
log.Printf("queued=%d sent=%d dropped=%d failed=%d sampled=%d queue_size=%d",
    stats.Enqueued, stats.Sent, stats.Dropped, stats.Failed, stats.Sampled,
    client.QueueSize(),
)
```

### Typed Errors

Distinguish retryable from permanent failures:

```go
err := client.SendSync(ctx, myErr)
if els.IsRetryableErr(err) {
    // 5xx, 429, network — safe to retry
} else {
    // 4xx — don't retry (auth, validation)
}
```

---

## Versioning

The `AppVersion` field powers ELS analytics for **regression detection** ("which errors first appeared in the latest release") and version-aware timelines.

ELS accepts **any string up to 128 characters** and auto-detects the format:

| Type | Examples |
|---|---|
| `date-compact` | `20260507120000` |
| `semver` | `1.2.3`, `1.0.0-rc.1`, `2.0.0+build.123` |
| `calver` | `2026.05`, `26.05.07` |
| `date-iso` | `2026-05-07`, `2026-05-07T12:00:00Z` |
| `git-sha` | `a1b2c3d`, `a1b2c3d4e5f6...` |
| `prefixed` | `v1.2.3`, `release-2026.05` |
| `opaque` | `production`, `nightly`, `customLabel` |

**Recommended setup**: pass build time via Dockerfile + CI:

```dockerfile
ARG BUILD_VERSION=dev
ENV BUILD_VERSION=$BUILD_VERSION
```

```yaml
# .gitlab-ci.yml
- export BUILD_VERSION=$(date -u +%Y%m%d%H%M%S)
- docker build --build-arg BUILD_VERSION="$BUILD_VERSION" ...
```

```go
els.Init(els.Config{ ..., AppVersion: os.Getenv("BUILD_VERSION") })
```

**Per-call override**: use `els.WithAppVersion(...)` to override per `Capture` call:

```go
els.CaptureErrorGlobal(err, els.WithAppVersion("client-app-v3"), els.WithMeta(map[string]any{...}))
```

---

## Configuration

```go
els.Config{
    // Required
    Endpoint string    // ELS API base URL
    APIKey   string    // API key

    // Identity (recommended)
    AppSlug       string // Application identifier
    DeploymentEnv string // DEV, STAGING, PRODUCTION
    ServiceName   string // Microservice name
    AppVersion    string // App version (any format up to 128 chars, see Versioning)

    // Batching
    BatchSize     int           // Max entries per request (default: 50)
    BatchInterval time.Duration // Flush interval (default: 5s)
    BufferSize    int           // In-memory queue capacity (default: 1000)

    // Retry
    MaxRetries     int           // Retry attempts (default: 3)
    RetryBaseDelay time.Duration // Initial delay, doubles each attempt (default: 1s)
    Timeout        time.Duration // HTTP timeout (default: 10s)

    // Buffering
    BufferDir         string // Disk buffer directory (default: os.TempDir())
    MaxBufferFileSize int64  // Max buffer file size (default: 100MB)

    // Filtering
    MinLevel   string  // Minimum level to capture (default: all)
    SampleRate float64 // 0.0-1.0, critical always passes (default: 1.0)

    // Hooks
    BeforeSend func(*ErrorEntry) *ErrorEntry // Filter/mutate before send
    OnError    func(error)                   // Internal error callback

    // Advanced
    HTTPClient   *http.Client // Custom HTTP client
    FlushTimeout time.Duration // Flush() max wait (default: 10s)
    DefaultLevel  string       // Default: "error"
    DefaultSource string       // Default: "server"
    Debug         bool         // Verbose logging to stderr
}
```

---

## Graceful Shutdown

```go
defer client.Close() // or defer els.Close() for global
```

`Close()` stops new captures → drains queue → sends remaining → buffers unsent to disk.

---

## Examples

- [Basic usage (EN)](examples/en/basic/main.go) — init, capture, sync send, health check
- [HTTP middleware (EN)](examples/en/middleware/main.go) — panic recovery, manual capture
- [Базовый пример (RU)](examples/ru/basic/main.go)
- [HTTP middleware (RU)](examples/ru/middleware/main.go)

## Field Reference

See [docs/FIELDS.md](docs/FIELDS.md) for all entry fields with descriptions.

## License

MIT
