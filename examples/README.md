# els-go examples

Runnable examples for the [ELS Go SDK](../README.md). Each folder is a
standalone module (its own `go.mod`) that builds against the local SDK.

> 🇷🇺 [Русская версия → README_RU.md](README_RU.md)

## Run

```bash
cd en/minimal-quickstart
ELS_API_KEY=els_live_xxxxxxxx go run .
```

English examples live in `en/`, Russian (translated comments) in `ru/` —
the code is identical.

## Scenarios

| Example | What it shows |
|---|---|
| `minimal-quickstart` | Smallest setup — only `APIKey` + `AppSlug` |
| `basic` | Errors, messages, sync send, graceful shutdown |
| `capture-error-vs-message` | `CaptureError` vs `CaptureMessage` vs `SendSync` |
| `level-shortcuts` | `Debug/Info/Warning/Error/Critical` helpers |
| `levels` | Typed `Level` + slog↔ELS mapping |
| `slog` | `log/slog` integration; errors captured with a stack trace |
| `context` | Propagate request/trace ID via `context.Context` |
| `service-defaults` | Set `ServiceName`/`AppSlug`/env once on the client |
| `filtering-sampling` | `MinLevel` + `SampleRate` (critical always passes) |
| `before-send-redaction` | `BeforeSend` hook to redact PII / drop entries |
| `worker-shutdown` | Batching + graceful shutdown on SIGINT/SIGTERM |
| `disk-buffer` | Offline resilience via on-disk buffer |
| `user-context` | Attach user info to every entry |
| `health-check` | `Health()` as a readiness probe |
| `global-facade` | Package-level `Init` + `*Global` helpers |
| `custom-http-client` | Provide a custom `*http.Client` |
| `middleware` | net/http panic-recovery middleware |
