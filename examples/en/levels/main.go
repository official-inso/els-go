// Example: typed levels and slog<->ELS level mapping.
//
// Level is a typed string with helper methods (String, Valid, Priority, ToSlog)
// and the package provides LevelFromSlog for the reverse direction.
package main

import (
	"fmt"
	"log/slog"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:   os.Getenv("ELS_API_KEY"),
		AppSlug:  "levels-demo",
		MinLevel: els.LevelInfo, // debug entries are dropped before send
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	levels := []els.Level{
		els.LevelDebug, els.LevelInfo, els.LevelWarning, els.LevelError, els.LevelCritical,
	}
	for _, lvl := range levels {
		fmt.Printf("%-8s valid=%v priority=%d slog=%v\n",
			lvl, lvl.Valid(), lvl.Priority(), lvl.ToSlog())
		client.CaptureMessage("level demo: "+lvl.String(), lvl)
	}

	// Map a slog.Level to the ELS level (useful in custom handlers).
	fmt.Println("slog.LevelError ->", els.LevelFromSlog(slog.LevelError))

	client.Flush()
}
