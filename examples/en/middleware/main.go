// Example: HTTP middleware usage with the ELS Go SDK.
//
// This example demonstrates how to use the built-in panic recovery
// middleware to automatically capture unhandled panics in HTTP handlers.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	els "github.com/official-inso/els-go"
)

func main() {
	// Initialize the ELS client
	client, err := els.New(els.Config{
		// Endpoint is hardcoded in the SDK — no need to configure it.
		APIKey:        os.Getenv("ELS_API_KEY"),
		AppSlug:       "web-api",
		DeploymentEnv: "PRODUCTION",
		ServiceName:   "user-service",
		BatchInterval: 2 * time.Second,
		OnError: func(err error) {
			log.Printf("[ELS] %v", err)
		},
	})
	if err != nil {
		log.Fatalf("Failed to initialize ELS: %v", err)
	}
	defer client.Close()

	// Set up HTTP routes
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("id")
		if userID == "" {
			// Manually capture a non-fatal error
			client.CaptureMessage("missing user ID in request", els.LevelWarning,
				els.WithURL(r.URL.String()),
				els.WithUserAgent(r.UserAgent()),
				els.WithReferrer(r.Referer()),
			)
			http.Error(w, "id parameter required", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": userID, "name": "John"})
	})

	mux.HandleFunc("/api/panic", func(w http.ResponseWriter, r *http.Request) {
		// This panic will be automatically captured by the middleware
		panic("unexpected nil pointer in user lookup")
	})

	// Use RecoverMiddleware — captures panics AND returns 500 response.
	// (Use client.Middleware() instead if you have your own recovery layer.)
	handler := client.RecoverMiddleware(mux)

	addr := ":8080"
	fmt.Printf("Server listening on %s\n", addr)
	fmt.Println("Try: GET /api/health, GET /api/users?id=1, GET /api/panic")
	log.Fatal(http.ListenAndServe(addr, handler))
}
