// Example: disk buffering for offline resilience.
//
// When the ELS server is unreachable, entries are persisted to BufferDir and
// flushed on the next successful send instead of being lost.
package main

import (
	"errors"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:            os.Getenv("ELS_API_KEY"),
		AppSlug:           "offline-demo",
		BufferDir:         "./.els-buffer",  // entries persisted here when offline
		MaxBufferFileSize: 50 * 1024 * 1024, // cap the on-disk buffer at 50 MB
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	client.CaptureError(errors.New("captured even if ELS is down right now"))
	client.Flush()
}
