# ELS Go SDK

A high-performance, asynchronous Go SDK for the [Error Logs Service (ELS)](https://github.com/official-inso/els-go).

[Русская версия](README_RU.md)

## Features

- **Zero external dependencies** — stdlib only
- **Asynchronous batching** — background goroutine collects entries and sends them in efficient batches
- **Automatic retries** — exponential backoff with 429 rate-limit handling
- **Disk buffering** — unsent entries are persisted to disk and retried on next startup
- **Panic recovery middleware** — automatic capture of HTTP handler panics
- **BeforeSend hook** — filter or mutate entries before they are sent
- **Session tracking** — automatic per-process session ID for error correlation

## Installation

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

    // Capture an error (async, non-blocking)
    client.CaptureError(errors.New("database connection timeout"),
        els.WithURL("/api/users"),
        els.WithLevel(els.LevelCritical),
    )

    // Capture an informational message
    client.CaptureMessage("service started successfully", els.LevelInfo,
        els.WithURL("/"),
    )
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
| `BufferDir` | string | `os.TempDir()` | Directory for disk buffer file |
| `BeforeSend` | func | `nil` | Hook to filter/mutate entries |
| `OnError` | func | `nil` | Internal error callback |
| `DefaultLevel` | string | `"error"` | Default severity level |
| `DefaultSource` | string | `"server"` | Default error source |
| `Debug` | bool | `false` | Enable debug logging |

## HTTP Middleware

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/health", healthHandler)

// Wrap with ELS panic recovery
http.ListenAndServe(":8080", client.Middleware(mux))
```

## Capture Options

Options can be passed to `CaptureError`, `CaptureMessage`, and `CaptureEntry`:

```go
client.CaptureError(err,
    els.WithLevel(els.LevelCritical),
    els.WithURL("/api/orders/123"),
    els.WithSource(els.SourceServer),
    els.WithUserAgent("CustomBot/1.0"),
    els.WithMeta(map[string]any{"orderId": "123", "userId": "456"}),
    els.WithSessionID("custom-session-id"),
)
```

## Disk Buffering

When the ELS server is unreachable after all retries, entries are saved to a `.els-buffer.jsonl` file in the configured `BufferDir` (defaults to the system temp directory). On the next client startup, the buffer is automatically flushed.

## Graceful Shutdown

Always call `client.Close()` before your application exits to ensure all pending entries are sent:

```go
client, _ := els.New(config)
defer client.Close()
```

`Close()` will:
1. Signal the background worker to stop accepting new entries
2. Drain and send all entries remaining in the queue
3. Persist any unsent entries to disk if sending fails

## Field Reference

See [docs/FIELDS.md](docs/FIELDS.md) for a detailed description of all error entry fields.

## Examples

- [Basic usage (EN)](examples/en/basic/main.go)
- [HTTP middleware (EN)](examples/en/middleware/main.go)
- [Базовый пример (RU)](examples/ru/basic/main.go)
- [HTTP middleware (RU)](examples/ru/middleware/main.go)

## License

MIT
