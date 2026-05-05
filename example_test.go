package els_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	els "github.com/official-inso/els-go"
)

func ExampleNew() {
	client, err := els.New(els.Config{
		Endpoint:      "https://api.example.com/els",
		APIKey:        "your-api-key",
		AppSlug:       "my-service",
		DeploymentEnv: "PRODUCTION",
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	fmt.Println("client initialized")
	// Output: client initialized
}

func ExampleClient_CaptureError() {
	client, _ := els.New(els.Config{
		Endpoint: "https://api.example.com/els",
		APIKey:   "key",
	})
	defer client.Close()

	// Async capture with options
	client.CaptureError(
		errors.New("connection timeout"),
		els.WithURL("/api/data"),
		els.WithLevel(els.LevelCritical),
		els.WithMeta(map[string]any{"retries": 3}),
	)
}

func ExampleClient_SendSync() {
	client, _ := els.New(els.Config{
		Endpoint: "https://api.example.com/els",
		APIKey:   "key",
	})
	defer client.Close()

	// Synchronous send — blocks until confirmed
	ctx := context.Background()
	err := client.SendSync(ctx, errors.New("critical payment failure"),
		els.WithURL("/api/payments"),
		els.WithLevel(els.LevelCritical),
	)
	if err != nil {
		fmt.Printf("send failed: %v\n", err)
	}
}

func ExampleClient_Middleware() {
	client, _ := els.New(els.Config{
		Endpoint: "https://api.example.com/els",
		APIKey:   "key",
	})
	defer client.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Panics in handlers are automatically captured as CRITICAL errors
	_ = client.RecoverMiddleware(mux)
}

func ExampleClient_Health() {
	client, _ := els.New(els.Config{
		Endpoint: "https://api.example.com/els",
		APIKey:   "key",
	})
	defer client.Close()

	ctx := context.Background()
	if err := client.Health(ctx); err != nil {
		fmt.Printf("ELS unreachable: %v\n", err)
	}
}
