// Example: basic usage of the ELS Go SDK.
//
// This example demonstrates how to initialize the client, capture errors
// and messages, and gracefully shut down.
package main

import (
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
		Endpoint:      os.Getenv("ELS_ENDPOINT"), // e.g., "https://api.example.com/els"
		APIKey:        os.Getenv("ELS_API_KEY"),
		AppSlug:       "my-service",
		DeploymentEnv: "PRODUCTION",
		ServiceName:   "api-gateway",

		// Optional: customize batching behavior
		BatchSize:     50,
		BatchInterval: 5 * time.Second,

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

	// --- Capture an error with automatic stack trace ---
	client.CaptureError(
		errors.New("database connection timeout"),
		els.WithURL("/api/users"),
		els.WithLevel(els.LevelCritical),
		els.WithMeta(map[string]any{
			"database": "postgres-primary",
			"timeout":  "30s",
		}),
	)

	// --- Capture an informational message ---
	client.CaptureMessage("service started successfully", els.LevelInfo,
		els.WithURL("/"),
	)

	// --- Capture a warning ---
	client.CaptureMessage("memory usage above 80%", els.LevelWarning,
		els.WithURL("/health"),
		els.WithMeta(map[string]any{"memoryPct": 82.5}),
	)

	// --- Capture a pre-built entry ---
	client.CaptureEntry(els.ErrorEntry{
		Message: "payment processing failed",
		URL:     "/api/payments/charge",
		Level:   els.LevelError,
		Meta: map[string]any{
			"orderId": "ord_12345",
			"amount":  9900,
		},
	})

	// Give the background worker time to send
	time.Sleep(time.Second)
	fmt.Println("All errors captured. Shutting down...")
}
