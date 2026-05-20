# ELS Go SDK

High-performance Go SDK for the **Error Logs Service (ELS)** — a managed SaaS for centralised event logging (debug → fatal) with AI-assisted error triage. Zero dependencies, async batching, disk buffering.

> 🇷🇺 [Русская версия → README_RU.md](README_RU.md)

## What you get

ELS ships with a built-in admin dashboard. Every event captured by this SDK lands there with full-text search, faceted filtering, AI-assisted diagnosis, and version-aware regression detection.

| | |
|---|---|
| ![Logs list](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/01-error-logs-list.png) | ![Event detail](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/02-event-detail-info.png) |
| Virtual table with facet sidebar (app, env, **version**, source, level, browser, IP, category). Live mode auto-refreshes every 5s. | Full event metadata: timestamps, geo, env, **app version**, fingerprint, session, repetition cards, in-session correlation. |
| ![AI diagnosis](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/03-error-detail-ai.png) | ![Analytics](https://raw.githubusercontent.com/official-inso/els-go/main/docs/screenshots/04-analytics-dashboard.png) |
| Parsed stack trace + AI-assisted diagnosis: what broke, where, how to fix. | Timeline, donuts, top URLs/IPs, hourly heatmap, **version-regression widget**. |

Don't have an API key yet? **[Sign up at lk.insoweb.ru](https://lk.insoweb.ru)** — takes under a minute.

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
    "os"

    els "github.com/official-inso/els-go"
)

func main() {
    // Option A: global client (recommended for most apps)
    if err := els.Init(els.Config{
        // Endpoint is hardcoded in the SDK — no need to configure it.
        APIKey:        os.Getenv("ELS_API_KEY"),
        AppSlug:       "my-service",
        DeploymentEnv: "PRODUCTION",
        AppVersion:    os.Getenv("BUILD_VERSION"), // see "Versioning" below
    }); err != nil {
        log.Fatal(err)
    }
    defer els.Close()

    els.CaptureErrorGlobal(errors.New("something broke"), els.WithURL("/api/users"))
}

// Option B: explicit client (for libraries or multiple instances)
func newClient() (*els.Client, error) {
    return els.New(els.Config{
        APIKey:  os.Getenv("ELS_API_KEY"),
        AppSlug: "my-service",
    })
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

Full list: `WithURL`, `WithLevel`, `WithSource`, `WithStack`, `WithMeta`, `WithUserAgent`, `WithLanguage`, `WithReferrer`, `WithSessionID`, `WithServiceName`, `WithComponentStack`, `WithAppVersion`, `WithRequest`, `WithCause`.

### Level Shortcuts

Logger-style helpers for messages — thin wrappers over `CaptureMessage`:

```go
client.Debug("cache warm")
client.Info("user logged in", els.WithMeta(map[string]any{"userId": 42}))
client.Warning("retry budget low")
client.Error("validation failed")   // a *message* at error level
client.Critical("disk full")
```

> `client.Error(msg string)` logs a message. To capture a Go `error` value
> **with a stack trace**, use `client.CaptureError(err)`.

### Context Propagation

Carry a request/trace ID through `context.Context` and attach it automatically:

```go
ctx = els.ContextWithRequestID(ctx, "req-123")
ctx = els.ContextWithTraceID(ctx, "trace-abc")

client.CaptureErrorCtx(ctx, err, els.WithURL("/api"))   // Meta gets requestId + traceId
client.CaptureMessageCtx(ctx, "step done", els.LevelInfo)
```

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

Route Go's standard `log/slog` to ELS with no call-site changes:

```go
logger := slog.New(els.SlogHandler(client, nil))
slog.SetDefault(logger)

slog.Info("cache warm", "keys", 1280) // attrs → entry Meta

// A record carrying an error attribute ("err"/"error") is captured WITH a full
// stack trace and the error text in Meta — just like CaptureError:
slog.Error("db query failed", "err", dbErr, "table", "users")
```

`ServiceName`/`AppSlug` from `Config` are attached automatically. Tune via
`SlogHandlerOptions`:

```go
els.SlogHandler(client, &els.SlogHandlerOptions{
    AddSource:           true,                 // file:line for non-error records
    CaptureStackOnError: true,                 // full stack on error records (default)
    ErrorKeys:           []string{"err", "error"}, // attrs treated as the cause
    URL:                 "slog",               // default URL for entries
})
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
    APIKey string // API key (endpoint is hardcoded in the SDK)

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
    MaxRetries     int           // Retry attempts (default: 3; set -1 to disable)
    RetryBaseDelay time.Duration // Initial delay, doubles each attempt (default: 1s)
    Timeout        time.Duration // HTTP timeout (default: 10s)

    // Buffering
    BufferDir         string // Disk buffer directory (default: os.TempDir())
    MaxBufferFileSize int64  // Max buffer file size (default: 100MB)

    // Filtering
    MinLevel   els.Level // Minimum level to capture (default: all)
    SampleRate float64   // 0.0-1.0, critical always passes (default: 1.0)

    // Throughput
    SenderConcurrency int // Parallel sender goroutines (default: 4)

    // Hooks
    BeforeSend func(*ErrorEntry) *ErrorEntry // Filter/mutate before send
    OnError    func(error)                   // Internal error callback

    // Advanced
    HTTPClient   *http.Client // Custom HTTP client
    FlushTimeout time.Duration // Flush() max wait (default: 10s)
    DefaultLevel  els.Level    // Default: "error"
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

## Migration

### From go.uber.org/zap

**Before:**

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

logger.Info("user logged in", zap.Int("userId", 42))
logger.Error("payment failed", zap.Error(err))
```

**After:**

```go
import els "github.com/official-inso/els-go"

els.Init(els.Config{
    APIKey:        os.Getenv("ELS_API_KEY"),
    AppSlug:       "my-service",
    DeploymentEnv: "PRODUCTION",
    AppVersion:    os.Getenv("BUILD_VERSION"),
})
defer els.Close()

els.CaptureMessageGlobal("user logged in", els.LevelInfo,
    els.WithMeta(map[string]any{"userId": 42}))
els.CaptureErrorGlobal(err, els.WithURL("/api/pay"))
```

| zap | ELS | Notes |
|---|---|---|
| `zap.NewProduction()` | `els.Init(els.Config{...})` | Once at startup |
| `logger.Info(msg, zap.X(...))` | `els.CaptureMessageGlobal(msg, level, els.WithMeta(...))` | Or use `els.SlogHandler` for `log/slog` idioms |
| `zap.Error(err)` | `els.CaptureErrorGlobal(err, ...)` | Dedicated path for errors |
| `logger.With(fields)` | `els.WithMeta(...)` per call | Or build a wrapper that pre-bakes meta |
| `logger.Sugar()` | Not provided | Stay structured |
| Sampling | `Config.SampleRate` | Same idea |

**Gotchas:**

- zap's encoder choices (`console`, `json`) are not exposed — the wire format is fixed JSON.
- For drop-in compatibility, prefer the `slog` integration (see Features) — zap's structured calls map naturally.

---

### From sirupsen/logrus

**Before:**

```go
import "github.com/sirupsen/logrus"

log := logrus.New()
log.SetFormatter(&logrus.JSONFormatter{})
log.SetLevel(logrus.InfoLevel)

log.WithFields(logrus.Fields{"userId": 42}).Info("user logged in")
log.WithError(err).Error("payment failed")
```

**After:**

```go
import els "github.com/official-inso/els-go"

els.Init(els.Config{
    APIKey:   os.Getenv("ELS_API_KEY"),
    AppSlug:  "my-service",
    MinLevel: els.LevelInfo,
})
defer els.Close()

els.CaptureMessageGlobal("user logged in", els.LevelInfo,
    els.WithMeta(map[string]any{"userId": 42}))
els.CaptureErrorGlobal(err, els.WithURL("/api/pay"))
```

| logrus | ELS | Notes |
|---|---|---|
| `logrus.New()` | `els.Init(els.Config{...})` | Single global or instance |
| `WithFields(Fields{...})` | `els.WithMeta(map[string]any{...})` | Same shape |
| `WithError(err)` | `els.CaptureErrorGlobal(err)` | Dedicated path |
| `SetLevel` | `Config.MinLevel` | Same idea |
| Hooks (`AddHook`) | `Config.BeforeSend` | Same role |
| Text formatter | Not provided | JSON only on the wire |

**Gotchas:**

- logrus is in maintenance mode. ELS aligns with modern `log/slog` — see Features for a structured handler.
- `logrus.PanicLevel` ≈ ELS `LevelCritical`; the SDK does not panic by itself.

---

### From log/slog (standard library)

`log/slog` integrates natively — no migration needed beyond pointing the handler at ELS.

**Before:**

```go
import "log/slog"
import "os"

logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
slog.SetDefault(logger)

slog.Info("user logged in", "userId", 42)
slog.Error("payment failed", "err", err)
```

**After:**

```go
import (
    "log/slog"
    els "github.com/official-inso/els-go"
)

client, _ := els.New(els.Config{
    APIKey:  os.Getenv("ELS_API_KEY"),
    AppSlug: "my-service",
})
defer client.Close()

logger := slog.New(els.SlogHandler(client, nil))
slog.SetDefault(logger)

slog.Info("user logged in", "userId", 42)
// slog.Error with an "err" attribute is captured WITH a full stack trace,
// just like CaptureError.
slog.Error("payment failed", "err", err)
```

| `log/slog` | ELS | Notes |
|---|---|---|
| `slog.NewJSONHandler(os.Stderr, nil)` | `els.SlogHandler(client, nil)` | Same handler contract |
| `slog.Info("msg", "k", v)` | unchanged | Routed to ELS as `info` event |
| `slog.With(...)` | unchanged | Adds key/value pairs to `meta` |
| Two destinations (stdout + ELS) | Compose multiple handlers | Wrap both via a fan-out handler |

**Gotchas:**

- The default level is `INFO`. Set `MinLevel` if you need to capture `DEBUG`.
- Stack traces from `slog.Error("msg", "err", err)` arrive as `meta.err` — to ship the full stack, use `els.CaptureError(err, ...)` directly for the error path.

---

## Examples

- [Basic usage (EN)](examples/en/basic/main.go) — init, capture, sync send, health check
- [HTTP middleware (EN)](examples/en/middleware/main.go) — panic recovery, manual capture
- [Базовый пример (RU)](examples/ru/basic/main.go)
- [HTTP middleware (RU)](examples/ru/middleware/main.go)

## Field Reference

See [docs/FIELDS.md](docs/FIELDS.md) for all entry fields with descriptions.

## Why ELS

ELS for Go is a focused logging SaaS, not a full observability suite. It optimises for capture speed, AI-driven triage, and a low integration cost.

- **Lower weight.** Single module, zero external dependencies.
- **Zero external API calls.** Only `POST /errors[/batch]` and `GET /health`.
- **AI-assisted diagnosis** on every stack trace, out of the box.
- **5-minute integration.** `go get` + `els.Init(...)` and you're done.
- **Predictable price.** Tariffs live in your personal cabinet.

### Detailed comparison

| Category | ELS | Sentry | Datadog / New Relic | Grafana Loki | LogRocket / Logtail / BetterStack |
|---|---|---|---|---|---|
| Hosting model | Managed SaaS | SaaS or self-hosted | SaaS only | Self-hosted / Grafana Cloud | SaaS |
| SDK runtime deps | Zero | Medium (sub-SDKs, integrations) | Heavy (agent + tracing) | Promtail / agent | Medium |
| Typical integration time | ~5 min | 10–20 min | 30–60 min | Hours to days | 10–20 min |
| AI-assisted triage | Built-in | Paid add-on | Paid add-on | None | None |
| Error grouping / fingerprint | Yes | Yes | Yes | Manual via LogQL | Partial |
| Source-map upload | No | Yes | Yes | n/a | Partial |
| Session replay (frontend) | No | Paid | Paid | n/a | Yes (core) |
| Distributed tracing / APM | No | Partial | Yes (core) | Yes with Tempo | No |
| Infrastructure metrics | No | No | Yes (core) | Yes with Mimir | No |
| Free tier log retention | 24 hours | 30 days (limited volume) | Trial only | Self-cost | 3–30 days |
| Russian-language support / docs | Native | Community | Limited | Community | None |

### When ELS is the wrong choice

- You need a single vendor for **APM + logs + metrics** under one bill — go Datadog or New Relic.
- Your frontend bug triage relies on **DOM session replay** — go LogRocket or Sentry Replay.
- You ship a **public mobile app** and need crash symbolication + ANR detection — Firebase Crashlytics or Sentry Mobile.

For everything else — backend errors, frontend JS errors, request logs, structured app events with version-aware analytics — ELS is built to be the cheapest path to a working dashboard.

→ **Sign up at [lk.insoweb.ru](https://lk.insoweb.ru)** to grab an API key.

## Other ELS SDKs

Same wire format, same dashboard — pick by stack.

**Go** (this repo)
- `github.com/official-inso/els-go` — core SDK with `slog` handler, HTTP middleware

**Node.js family**
- [`@inso_web/els-client`](https://github.com/official-inso/els-client) — base TS / Node / browser client
- [`@inso_web/els-express`](https://github.com/official-inso/els-express) — Express middleware
- [`@inso_web/els-next`](https://github.com/official-inso/els-next) — Next.js helpers
- [`@inso_web/els-nest`](https://github.com/official-inso/els-nest) — NestJS module
- [`@inso_web/els-react`](https://github.com/official-inso/els-react) — React Provider, hooks, ErrorBoundary
- [`@inso_web/els-vue`](https://github.com/official-inso/els-vue) — Vue 3 plugin

**Other stacks**
- [`Inso.Els`](https://github.com/official-inso/els-csharp) — .NET (Core + ASP.NET Core + ILogger)
- [`io.github.official-inso:els-core`](https://github.com/official-inso/els-java) — Java + Spring Boot starter + SLF4J

## Pricing

Free tier — **24-hour log retention**. See **[lk.insoweb.ru](https://lk.insoweb.ru)** for the full tariff matrix.

## License

MIT
