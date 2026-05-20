// Package els provides an asynchronous, high-performance Go SDK for the
// Error Logs Service (ELS). It captures errors and messages, batches them
// in a background goroutine, and delivers them to the ELS HTTP API with
// automatic retries and disk-based buffering for resilience.
//
// Key features:
//   - Zero external dependencies (stdlib only)
//   - Asynchronous batching with a parallel sender pool, so a slow or
//     unreachable server never blocks ingestion (see Config.SenderConcurrency)
//   - Synchronous send via SendSync for critical errors requiring confirmation
//   - Level shortcuts: Debug, Info, Warning, Error, Critical
//   - log/slog integration (SlogHandler); an slog.Error carrying an "err"
//     attribute is captured with a full stack trace, like CaptureError
//   - Context propagation: ContextWithRequestID / ContextWithTraceID +
//     CaptureErrorCtx / CaptureMessageCtx
//   - Automatic retry with exponential backoff and 429 rate-limit handling
//     (set MaxRetries to a negative value to disable retries)
//   - Typed errors (SendError) distinguishing retryable from permanent failures
//   - Disk buffer for offline resilience (entries survive process restarts)
//   - Sampling (SampleRate) — critical entries always pass
//   - Level filtering via MinLevel to drop low-severity entries
//   - User context (SetUser) attached to every entry
//   - net/http middleware for automatic panic recovery (Middleware and RecoverMiddleware)
//   - BeforeSend hook for filtering or mutating entries
//   - Graceful shutdown with queue draining; Health check; runtime Stats
//
// Quick start:
//
//	client, err := els.New(els.Config{
//	    APIKey:        "your-api-key",
//	    AppSlug:       "my-service",
//	    DeploymentEnv: "PRODUCTION",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Async (non-blocking)
//	client.CaptureError(errors.New("something went wrong"), els.WithURL("/api"))
//
//	// Sync (blocks until confirmed)
//	err = client.SendSync(ctx, errors.New("critical failure"), els.WithURL("/api"))
package els
