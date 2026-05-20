// Example: basic usage of the ELS Go SDK.
//
// This example demonstrates how to initialize the client, capture errors
// and messages, use synchronous send for critical errors, and gracefully shut down.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	els "github.com/official-inso/els-go"
)

func main() {
	// Initialize the ELS client
	client, err := els.New(els.Config{
		// Endpoint is hardcoded in the SDK — no need to configure it.
		APIKey:        os.Getenv("ELS_API_KEY"),
		AppSlug:       "my-service",
		DeploymentEnv: "PRODUCTION",
		ServiceName:   "api-gateway",

		// Optional: customize batching behavior
		BatchSize:     50,
		BatchInterval: 5 * time.Second,

		// Optional: minimum level filter (drops debug and info)
		MinLevel: els.LevelWarning,

		// Optional: error callback for debugging
		OnError: func(err error) {
			log.Printf("[ELS] internal error: %v", err)
		},
	})
	if err != nil {
		log.Fatalf("Failed to initialize ELS: %v", err)
	}
	// Always close the client to flush pending entries
	defer client.Close()

	// --- Check server connectivity ---
	ctx := context.Background()
	if err := client.Health(ctx); err != nil {
		log.Printf("ELS server not reachable: %v (entries will be buffered)", err)
	}

	// --- Capture an error with automatic stack trace (async) ---
	client.CaptureError(
		errors.New("database connection timeout"),
		els.WithURL("/api/users"),
		els.WithLevel(els.LevelCritical),
		els.WithMeta(map[string]any{
			"database": "postgres-primary",
			"timeout":  "30s",
		}),
	)

	// --- Capture a warning (async) ---
	client.CaptureMessage("memory usage above 80%", els.LevelWarning,
		els.WithURL("/health"),
		els.WithMeta(map[string]any{"memoryPct": 82.5}),
	)

	// --- Synchronous send for critical errors (blocks until confirmed) ---
	err = client.SendSync(ctx, errors.New("payment processing failed"),
		els.WithURL("/api/payments/charge"),
		els.WithLevel(els.LevelCritical),
		els.WithMeta(map[string]any{
			"orderId": "ord_12345",
			"amount":  9900,
		}),
	)
	if err != nil {
		// Check if error is retryable
		if els.IsRetryableErr(err) {
			log.Printf("Retryable ELS error: %v", err)
		} else {
			log.Printf("Permanent ELS error: %v", err)
		}
	}

	// --- Capture a pre-built entry (async) ---
	client.CaptureEntry(els.ErrorEntry{
		Message: "unusual traffic pattern detected",
		URL:     "/api/analytics",
		Level:   els.LevelWarning,
		Meta: map[string]any{
			"rps":       15000,
			"threshold": 10000,
		},
	})

	// Give the background worker time to send
	client.Flush()
	fmt.Println("All errors captured. Shutting down...")
}
