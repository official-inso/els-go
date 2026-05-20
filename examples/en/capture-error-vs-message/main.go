// Example: CaptureError vs CaptureMessage vs SendSync.
package main

import (
	"context"
	"errors"
	"log"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:  os.Getenv("ELS_API_KEY"),
		AppSlug: "capture-demo",
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// 1) CaptureError — async, automatic stack trace from the call site.
	client.CaptureError(errors.New("upstream timeout"),
		els.WithURL("/api/orders"),
		els.WithLevel(els.LevelError),
	)

	// 2) CaptureMessage — async, no stack, explicit level.
	client.CaptureMessage("retrying request", els.LevelWarning,
		els.WithMeta(map[string]any{"attempt": 2}),
	)

	// 3) SendSync — blocks until delivered. Use for critical events you can't lose.
	ctx := context.Background()
	if err := client.SendSync(ctx, errors.New("payment capture failed"),
		els.WithURL("/api/payments"),
		els.WithLevel(els.LevelCritical),
	); err != nil {
		if els.IsRetryableErr(err) {
			log.Printf("retryable ELS error: %v", err)
		} else {
			log.Printf("permanent ELS error: %v", err)
		}
	}

	client.Flush()
}
