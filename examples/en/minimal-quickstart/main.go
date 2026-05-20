// Example: the smallest possible ELS integration.
//
// Only APIKey and AppSlug are required — the endpoint is hardcoded in the SDK.
package main

import (
	"log"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:  os.Getenv("ELS_API_KEY"),
		AppSlug: "quickstart",
	})
	if err != nil {
		log.Fatalf("init ELS: %v", err)
	}
	defer client.Close()

	client.CaptureMessage("hello from els-go", els.LevelInfo)
	client.Flush()
}
