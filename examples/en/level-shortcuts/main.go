// Example: logger-style level shortcuts.
//
// Debug/Info/Warning/Error/Critical are thin wrappers over CaptureMessage.
package main

import (
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:  os.Getenv("ELS_API_KEY"),
		AppSlug: "shortcuts-demo",
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	client.Debug("verbose detail")
	client.Info("user logged in", els.WithMeta(map[string]any{"userId": 42}))
	client.Warning("retry budget low")
	client.Error("validation failed") // a *message* at error level
	client.Critical("disk full")

	// To capture a Go error value WITH a stack trace, use CaptureError instead:
	//   client.CaptureError(err)

	client.Flush()
}
