# ELS Go SDK

A high-performance, asynchronous Go SDK for the [Error Logs Service (ELS)](https://github.com/official-inso/els-go).

[Русская версия](README_RU.md)

## Features

- **Zero external dependencies** — stdlib only
- **Asynchronous batching** — background goroutine collects entries and sends them in efficient batches
- **Synchronous send** — `SendSync()` for critical errors that must be confirmed
- **Automatic retries** — exponential backoff with 429 rate-limit handling
- **Typed errors** — distinguish retryable vs permanent failures with `IsRetryableErr()`
- **Disk buffering** — unsent entries are persisted to disk and retried on next startup
- **Level filtering** — drop low-severity entries in production with `MinLevel`
- **Panic recovery middleware** — automatic capture of HTTP handler panics
- **Health check** — verify ELS server connectivity before operations
- **BeforeSend hook** — filter or mutate entries before they are sent
- **Session tracking** — automatic per-process session ID for error correlation
- **Graceful shutdown** — `Close()` drains queue and persists remaining entries

## Installation

```bash
go get github.com/official-inso/els-go
```

## Quick Start

```go
package main

import (
    "context"
    "errors"
    "log"

    els "github.com/official-inso/els-go"
)

func main() {
    client, err := els.New(els.Config{
        Endpoint:      "https://api.example.com/els",
        APIKey:        "your-api-key",
        AppSlug:       "my-service",
        DeploymentEnv: "PRODUCTION",
        ServiceName:   "api-gateway",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Async capture (non-blocking)
    client.CaptureError(errors.New("something went wrong"),
        els.WithURL("/api/users"),
        els.WithLevel(els.LevelError),
    )

    // Sync send for critical errors (blocks until confirmed)
    ctx := context.Background()
    if err := client.SendSync(ctx, errors.New("payment failed"),
        els.WithURL("/api/pay"),
        els.WithLevel(els.LevelCritical),
    ); err != nil {
        log.Printf("failed to report: %v", err)
    }
}
```

## Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Endpoint` | string | *required* | Base URL of the ELS API |
| `APIKey` | string | *required* | API key for authentication |
| `AppSlug` | string | `""` | Application identifier |
| `DeploymentEnv` | string | `""` | Environment (DEV, STAGING, PRODUCTION) |
| `ServiceName` | string | `""` | Microservice name |
| `BatchSize` | int | `50` | Max entries per batch request |
| `BatchInterval` | Duration | `5s` | Max time before flushing partial batch |
| `BufferSize` | int | `1000` | In-memory queue capacity |
| `MaxRetries` | int | `3` | Retry attempts for failed requests |
| `RetryBaseDelay` | Duration | `1s` | Initial retry delay (doubles each attempt) |
| `Timeout` | Duration | `10s` | HTTP request timeout |
| `FlushTimeout` | Duration | `10s` | Max time for Flush() to wait |
| `BufferDir` | string | `os.TempDir()` | Directory for disk buffer file |
| `MaxBufferFileSize` | int64 | `100MB` | Max disk buffer size before dropping |
| `MinLevel` | string | `""` | Minimum level to capture (filters lower) |
| `BeforeSend` | func | `nil` | Hook to filter/mutate entries |
| `OnError` | func | `nil` | Internal error callback |
| `DefaultLevel` | string | `"error"` | Default severity level |
| `DefaultSource` | string | `"server"` | Default error source |
| `Debug` | bool | `false` | Enable debug logging |

## Synchronous Send

For critical errors where delivery must be confirmed (payments, auth failures):

```go
err := client.SendSync(ctx, errors.New("payment failed"),
    els.WithURL("/api/payments"),
    els.WithLevel(els.LevelCritical),
)
if err != nil {
    if els.IsRetryableErr(err) {
        // Server/network issue — safe to retry later
    } else {
        // Permanent error (auth, validation) — don't retry
    }
}
```

## Level Filtering

Drop low-severity entries in production:

```go
client, _ := els.New(els.Config{
    // ...
    MinLevel: els.LevelWarning, // drops debug and info
})
```

Level priority: `debug` < `info` < `warning` < `error` < `critical`

## Health Check

```go
if err := client.Health(ctx); err != nil {
    log.Printf("ELS unreachable: %v", err)
}
```

## HTTP Middleware

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/health", healthHandler)

// Option 1: Captures panic and re-raises (use with your own recovery)
handler := client.Middleware(mux)

// Option 2: Captures panic and returns 500 (standalone)
handler := client.RecoverMiddleware(mux)

http.ListenAndServe(":8080", handler)
```

## Capture Options

```go
client.CaptureError(err,
    els.WithLevel(els.LevelCritical),
    els.WithURL("/api/orders/123"),
    els.WithSource(els.SourceServer),
    els.WithUserAgent("CustomBot/1.0"),
    els.WithLanguage("en-US"),
    els.WithReferrer("http://example.com"),
    els.WithMeta(map[string]any{"orderId": "123"}),
    els.WithSessionID("custom-session-id"),
    els.WithServiceName("payment-worker"),
    els.WithStack(customStackTrace),
    els.WithComponentStack(reactTrace),
)
```

## Disk Buffering

When the ELS server is unreachable after all retries, entries are saved to `.els-buffer.jsonl` in `BufferDir`. On next client startup, the buffer is automatically flushed. The file is capped at `MaxBufferFileSize` (default 100MB) — when exceeded, new entries are dropped.

## Graceful Shutdown

```go
client, _ := els.New(config)
defer client.Close()
```

`Close()` will:
1. Stop accepting new entries
2. Drain and send all entries remaining in the queue
3. Persist any unsent entries to disk

## Typed Errors

All send operations return `*SendError` which distinguishes retryable from permanent failures:

```go
err := client.SendSync(ctx, myErr)
var sendErr *els.SendError
if els.As(err, &sendErr) {
    fmt.Printf("Status: %d, Retryable: %v\n", sendErr.StatusCode, sendErr.IsRetryable)
}
```

## Field Reference

See [docs/FIELDS.md](docs/FIELDS.md) for a detailed description of all error entry fields.

## Examples

- [Basic usage (EN)](examples/en/basic/main.go)
- [HTTP middleware (EN)](examples/en/middleware/main.go)
- [Basic usage (RU)](examples/ru/basic/main.go)
- [HTTP middleware (RU)](examples/ru/middleware/main.go)

## License

MIT
