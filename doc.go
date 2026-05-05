// Package els provides an asynchronous, high-performance Go SDK for the
// Error Logs Service (ELS). It captures errors and messages, batches them
// in a background goroutine, and delivers them to the ELS HTTP API with
// automatic retries and disk-based buffering for resilience.
//
// Key features:
//   - Zero external dependencies (stdlib only)
//   - Asynchronous batching with configurable interval and batch size
//   - Automatic retry with exponential backoff and 429 rate-limit handling
//   - Disk buffer for offline resilience (entries survive process restarts)
//   - Graceful shutdown with queue draining
//   - net/http middleware for automatic panic recovery
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
//	client.CaptureError(errors.New("something went wrong"))
package els
