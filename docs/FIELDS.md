# ELS Error Entry Fields

All fields sent to the ELS API by the Go SDK.

## Required

| Field | Type | Max Length | Description |
|-------|------|-----------|-------------|
| `message` | string | 10,000 | Error message text |
| `url` | string | 2,000 | URL where the error occurred. Use `WithURL()` or `WithRequest(r)` |

## Auto-Filled

These are populated by the SDK — you don't set them manually.

| Field | Default Source | Description |
|-------|---------------|-------------|
| `timestamp` | `time.Now().UTC()` | ISO 8601 (RFC3339Nano) |
| `level` | `Config.DefaultLevel` | Severity: critical/error/warning/info/debug |
| `source` | `Config.DefaultSource` | Origin: server/client |
| `appSlug` | `Config.AppSlug` | Application identifier |
| `deploymentEnv` | `Config.DeploymentEnv` | Normalized server-side (dev→DEV, prod→PRODUCTION) |
| `serviceName` | `Config.ServiceName` | Microservice name |
| `sessionId` | Auto-generated | Per-process session for error correlation |
| `stack` | `runtime.Callers` | Stack trace (only for `CaptureError`) |

## Optional

Set via options (`WithX()`) or directly on `ErrorEntry`:

| Field | Type | Max | Option | Description |
|-------|------|-----|--------|-------------|
| `stack` | string | 50,000 | `WithStack(s)` | Override auto-captured stack |
| `componentStack` | string | 50,000 | `WithComponentStack(s)` | Framework component trace |
| `userAgent` | string | 1,000 | `WithUserAgent(ua)` | Client user-agent |
| `language` | string | 20 | `WithLanguage(l)` | Client locale (e.g., "en-US") |
| `screenSize` | string | 20 | — | Screen "WxH" (client-only) |
| `viewportSize` | string | 20 | — | Viewport "WxH" (client-only) |
| `referrer` | string | 2,000 | `WithReferrer(r)` | HTTP Referer |
| `meta` | object | — | `WithMeta(m)` | Arbitrary key-value data |

## Convenience Options

| Option | What it does |
|--------|-------------|
| `WithRequest(r *http.Request)` | Extracts URL, UserAgent, Referrer, Language, plus adds `http.method`, `http.host`, `http.remoteAddr`, `http.requestId` to Meta |
| `WithCause(err)` | Walks `Unwrap()` chain, stores cause messages in `meta["error.causes"]` |

## Level Values

| Value | Constant | When to use |
|-------|----------|-------------|
| `critical` | `els.LevelCritical` | System down, data loss |
| `error` | `els.LevelError` | Operation failed |
| `warning` | `els.LevelWarning` | Potential issue |
| `info` | `els.LevelInfo` | Significant event |
| `debug` | `els.LevelDebug` | Diagnostic detail |

## Source Values

| Value | Constant | Description |
|-------|----------|-------------|
| `server` | `els.SourceServer` | Backend/server error |
| `client` | `els.SourceClient` | Frontend/client error |

## Environment Normalization

The server normalizes `deploymentEnv` case-insensitively:

| You send | Stored as |
|----------|-----------|
| dev, development, test | `DEV` |
| staging, stage, stg | `STAGING` |
| prod, production | `PRODUCTION` |
| anything else | UPPERCASED |

## Server-Generated Fields

These are computed server-side (you cannot set them):

| Field | Description |
|-------|-------------|
| `traceId` | Unique identifier (format: `SRV-<timestamp>-<random>`) |
| `browser` | Parsed from userAgent |
| `urlPath` | Normalized path from url (UUIDs → `:id`) |
| `errorCategory` | Auto-categorized from message |
| `fingerprint` | Hash of message + stack + source |
| `ip` | Client IP from request |

## User Context Fields

When `SetUser()` is called, these appear in Meta automatically:

| Meta Key | Source |
|----------|--------|
| `user.id` | `UserContext.ID` |
| `user.email` | `UserContext.Email` |
| `user.name` | `UserContext.Name` |
| `user.<key>` | `UserContext.Extra[key]` |
