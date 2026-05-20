// Example: package-level (global) facade — init once, capture anywhere.
//
// Useful for small services and CLIs where threading a *Client through every
// function is inconvenient.
package main

import (
	"context"
	"errors"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	if err := els.Init(els.Config{
		APIKey:      os.Getenv("ELS_API_KEY"),
		AppSlug:     "global-demo",
		ServiceName: "cron",
	}); err != nil {
		panic(err)
	}
	defer els.Close()

	// No client to pass around — use the package-level helpers anywhere.
	els.CaptureMessageGlobal("nightly job started", els.LevelInfo)
	els.CaptureErrorGlobal(errors.New("nightly job failed"), els.WithURL("/cron/nightly"))
	_ = els.SendSyncGlobal(context.Background(), errors.New("critical cron failure"),
		els.WithLevel(els.LevelCritical))

	els.FlushGlobal()
}
