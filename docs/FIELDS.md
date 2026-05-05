# ELS Error Entry Fields

This document describes all fields of an error entry sent to the ELS API.

## Required Fields

| Field | Type | Description | Source |
|-------|------|-------------|--------|
| `message` | string | Error message text. Max 10,000 characters. | Your error: `err.Error()` or custom message |
| `url` | string | URL where the error occurred. Max 2,000 characters. | Request URL or page URL. Use `WithURL()` option |
| `timestamp` | string | ISO 8601 timestamp (RFC3339). | Auto-filled by SDK with current UTC time |

## Auto-Filled Fields

These fields are populated automatically by the SDK. You don't need to set them.

| Field | Type | Description | How it's filled |
|-------|------|-------------|-----------------|
| `timestamp` | string | When the error occurred | `time.Now().UTC().Format(time.RFC3339Nano)` |
| `sessionId` | string | Process-level session ID for error correlation | Auto-generated on client creation (`els-<hex>`) |
| `appSlug` | string | Application identifier | From `Config.AppSlug` |
| `deploymentEnv` | string | Deployment environment | From `Config.DeploymentEnv` |
| `serviceName` | string | Microservice name | From `Config.ServiceName` |
| `level` | string | Severity level | From `Config.DefaultLevel` (default: `"error"`) |
| `source` | string | Error origin | From `Config.DefaultSource` (default: `"server"`) |
| `stack` | string | Stack trace | Auto-captured via `runtime.Callers` for `CaptureError()` |

## Optional Fields

| Field | Type | Description | When to use |
|-------|------|-------------|-------------|
| `stack` | string | Stack trace. Max 50,000 chars. | Auto-captured for `CaptureError()`. Use `WithStack()` to override |
| `componentStack` | string | Framework component trace (e.g., React) | Frontend errors only. Use `WithComponentStack()` |
| `userAgent` | string | HTTP User-Agent header. Max 1,000 chars. | `r.UserAgent()` from HTTP request. Use `WithUserAgent()` |
| `language` | string | Client locale (e.g., "en-US"). Max 20 chars. | `r.Header.Get("Accept-Language")`. Use `WithLanguage()` |
| `screenSize` | string | Screen dimensions ("1920x1080"). Max 20 chars. | Client-side only |
| `viewportSize` | string | Viewport dimensions. Max 20 chars. | Client-side only |
| `referrer` | string | HTTP Referer header. Max 2,000 chars. | `r.Referer()`. Use `WithReferrer()` |
| `meta` | map | Arbitrary key-value metadata | Any additional context. Use `WithMeta()` |

## Level Values

| Value | When to use |
|-------|-------------|
| `critical` | System is down or data loss occurred |
| `error` | Operation failed but system continues |
| `warning` | Potential issue that didn't cause failure |
| `info` | Significant event (startup, config change) |
| `debug` | Detailed diagnostic information |

## Source Values

| Value | Description |
|-------|-------------|
| `server` | Error originated on the server/backend |
| `client` | Error originated on the client/frontend |

## DeploymentEnv Values

The server normalizes environment values case-insensitively:

| Input (any case) | Stored as |
|-------------------|-----------|
| `dev`, `development`, `test` | `DEV` |
| `staging`, `stage`, `stg` | `STAGING` |
| `prod`, `production` | `PRODUCTION` |
| Any other value | Uppercased as-is |

## Notes

- `traceId` is generated server-side and cannot be set by the client
- `message` and `url` are the only truly required fields — all others have defaults or are optional
- The server computes additional derived fields: `browser` (from userAgent), `urlPath` (from url), `errorCategory` (from message), `fingerprint` (from message+stack+source), `ip` (from request)
