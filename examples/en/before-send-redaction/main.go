// Example: BeforeSend hook to redact PII or drop entries.
//
// BeforeSend runs for every entry just before it is enqueued. Return nil to
// drop the entry, or mutate it in place to scrub sensitive data.
package main

import (
	"errors"
	"os"
	"strings"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:  os.Getenv("ELS_API_KEY"),
		AppSlug: "redaction-demo",
		BeforeSend: func(e *els.ErrorEntry) *els.ErrorEntry {
			// Drop debug noise entirely.
			if e.Level == els.LevelDebug {
				return nil
			}
			// Redact emails from the message.
			if strings.Contains(e.Message, "@") {
				e.Message = "[redacted email]"
			}
			// Strip a sensitive meta key.
			delete(e.Meta, "password")
			return e
		},
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	client.CaptureError(errors.New("login failed for user@example.com"),
		els.WithMeta(map[string]any{"password": "secret", "ip": "1.2.3.4"}),
	)
	client.CaptureMessage("verbose trace", els.LevelDebug) // dropped by BeforeSend

	client.Flush()
}
