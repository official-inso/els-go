// Package els provides an asynchronous, high-performance Go SDK for the
// Error Logs Service (ELS). It captures errors and messages, batches them
// in a background goroutine, and delivers them to the ELS HTTP API with
// automatic retries and disk-based buffering for resilience.
//
// Key features:
//   - Zero external dependencies (stdlib only)
//   - Asynchronous batching with configurable interval and batch size
//   - Synchronous send via SendSync for critical errors requiring confirmation
//   - Automatic retry with exponential backoff and 429 rate-limit handling
//   - Typed errors (SendError) distinguishing retryable from permanent failures
//   - Disk buffer for offline resilience (entries survive process restarts)
//   - Configurable max buffer file size to prevent unbounded disk usage
//   - Level filtering via MinLevel to drop low-severity entries
//   - Graceful shutdown with queue draining
//   - net/http middleware for automatic panic recovery (Middleware and RecoverMiddleware)
//   - Health check endpoint verification
//   - BeforeSend hook for filtering or mutating entries
//   - Process-level session ID for correlating related errors
//
// Quick start:
//
//	client, err := els.New(els.Config{
//	    Endpoint:      "https://api.example.com/els",
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
