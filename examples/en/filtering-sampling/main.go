// Example: MinLevel filtering and SampleRate.
//
// MinLevel drops low-severity entries before sending. SampleRate sends only a
// fraction of non-critical entries. Critical entries always bypass sampling.
package main

import (
	"fmt"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:     os.Getenv("ELS_API_KEY"),
		AppSlug:    "sampling-demo",
		MinLevel:   els.LevelWarning, // debug/info dropped before send
		SampleRate: 0.1,              // ~10% of non-critical entries are sent
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	for i := 0; i < 100; i++ {
		client.CaptureMessage("noisy warning", els.LevelWarning)
	}
	// Critical bypasses sampling — always delivered.
	client.CaptureMessage("disk failure", els.LevelCritical)

	client.Flush()
	fmt.Printf("stats: %+v\n", client.GetStats())
}
